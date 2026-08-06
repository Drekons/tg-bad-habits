package bot

import (
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/drek/tg-bad-habbits/internal/models"
	"github.com/drek/tg-bad-habbits/internal/repository"
	"github.com/drek/tg-bad-habbits/internal/service"
)

const dtLayout = "02.01.2006 15:04"

// Handler processes incoming Telegram updates.
type Handler struct {
	bot         *tgbotapi.BotAPI
	states      *StateManager
	userRepo    *repository.UserRepo
	habitRepo   *repository.HabitRepo
	relapseRepo *repository.RelapseRepo
	habitSvc    *service.HabitService
	statsSvc    *service.StatsService
}

func NewHandler(
	bot *tgbotapi.BotAPI,
	states *StateManager,
	userRepo *repository.UserRepo,
	habitRepo *repository.HabitRepo,
	relapseRepo *repository.RelapseRepo,
	habitSvc *service.HabitService,
	statsSvc *service.StatsService,
) *Handler {
	return &Handler{
		bot:         bot,
		states:      states,
		userRepo:    userRepo,
		habitRepo:   habitRepo,
		relapseRepo: relapseRepo,
		habitSvc:    habitSvc,
		statsSvc:    statsSvc,
	}
}

// Handle dispatches text messages (/start, habit wizard, legacy start/create buttons).
func (h *Handler) Handle(update tgbotapi.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)

	if text == "/start" {
		h.handleStart(msg)
		return
	}

	state := h.states.GetState(userID)

	switch state {
	case StateWaitStart:
		if text == "▶️ Нажмите чтобы начать" {
			h.handleRegistrationConfirm(msg)
		}

	case StateHabitName, StateHabitLastRelapse, StateHabitCost, StateHabitAvgCount, StateHabitAvgPeriod:
		h.handleHabitCreationStep(msg, state)

	default:
		switch {
		case text == "➕ Добавить привычку" || text == "➕ Создать первую вредную привычку":
			h.startHabitCreationByChat(msg.Chat.ID, userID)
		case text != "":
			// Навигация/срывы — только inline. Любой текст → главная.
			h.goMain(msg.Chat.ID, userID)
		}
	}
}

// HandleCallbackQuery — все действия с ID в data.
func (h *Handler) HandleCallbackQuery(cq *tgbotapi.CallbackQuery) {
	userID := cq.From.ID
	chatID := cq.Message.Chat.ID
	data := cq.Data
	cqMsgID := 0
	if cq.Message != nil {
		cqMsgID = cq.Message.MessageID
	}

	editID := h.overlayEditID(userID, cqMsgID)

	// Пустой answer по умолчанию; relapse_yes отвечает сам (toast в чат не пишет).
	answered := false
	defer func() {
		if answered {
			return
		}
		if _, err := h.bot.Request(tgbotapi.NewCallback(cq.ID, "")); err != nil {
			log.Printf("HandleCallbackQuery Answer: %v", err)
		}
	}()

	switch {
	case data == "main_menu":
		h.showMainMenuScreen(chatID, userID, editID)

	case data == "go_main":
		h.goMain(chatID, userID)

	case data == "add_habit":
		h.startHabitCreationByChat(chatID, userID)

	case strings.HasPrefix(data, "habit_menu:"):
		habitID, ok := h.parseOwnedHabitID(userID, strings.TrimPrefix(data, "habit_menu:"))
		if !ok {
			return
		}
		h.showHabitMenu(chatID, userID, habitID, editID)

	case strings.HasPrefix(data, "habit_stats:"):
		habitID, ok := h.parseOwnedHabitID(userID, strings.TrimPrefix(data, "habit_stats:"))
		if !ok {
			return
		}
		h.showHabitStatsByID(chatID, userID, habitID, editID)

	case strings.HasPrefix(data, "relapse_yes:"):
		habitID, ok := h.parseOwnedHabitID(userID, strings.TrimPrefix(data, "relapse_yes:"))
		if !ok {
			return
		}
		answered = true
		h.confirmRelapse(chatID, userID, habitID, cq.ID)

	case strings.HasPrefix(data, "relapse_no:"):
		rest := strings.TrimPrefix(data, "relapse_no:")
		parts := strings.Split(rest, ":")
		habitID, ok := h.parseOwnedHabitID(userID, parts[0])
		if !ok {
			return
		}
		back := "main"
		if len(parts) > 1 {
			back = parts[1]
		}
		h.declineRelapse(chatID, userID, habitID, back, editID)

	case strings.HasPrefix(data, "relapse:"):
		rest := strings.TrimPrefix(data, "relapse:")
		parts := strings.Split(rest, ":")
		habitID, ok := h.parseOwnedHabitID(userID, parts[0])
		if !ok {
			return
		}
		back := "main"
		if len(parts) > 1 {
			back = parts[1]
		}
		h.askConfirmRelapseByID(chatID, userID, habitID, back, editID)
	}
}

// overlayEditID: не редактируем главное сообщение (его трогает Updater) — только оверлеи.
func (h *Handler) overlayEditID(userID int64, cqMsgID int) int {
	if cqMsgID == 0 {
		return 0
	}
	_, mainID, ok, err := h.userRepo.GetMainMessage(userID)
	if err != nil {
		log.Printf("overlayEditID: %v", err)
	}
	if ok && mainID == cqMsgID {
		return 0
	}
	return cqMsgID
}

func (h *Handler) parseOwnedHabitID(userID int64, idStr string) (int64, bool) {
	habitID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, false
	}
	habit, getErr := h.habitRepo.GetByID(habitID)
	if getErr != nil || habit == nil || habit.UserID != userID {
		return 0, false
	}
	return habitID, true
}

func (h *Handler) handleStart(msg *tgbotapi.Message) {
	userID := msg.From.ID
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("handleStart GetByID: %v", err)
		return
	}

	if user != nil {
		habits, _ := h.habitRepo.GetByUserID(userID)
		h.states.SetState(userID, StateIdle)
		// После очистки истории чата edit старого main_message_id «успешен», но юзер его не видит.
		_ = h.userRepo.ClearMainMessage(userID)
		h.states.SetMainMessageID(userID, 0)
		h.sendHTML(msg.Chat.ID, "С возвращением! 👋", nil)
		if len(habits) > 0 {
			h.goMain(msg.Chat.ID, userID)
		} else {
			h.send(msg.Chat.ID, "У вас нет привычек. Создайте первую!", createFirstHabitKeyboard())
		}
		return
	}

	newUser := &models.User{ID: userID, Username: msg.From.UserName}
	if err := h.userRepo.Create(newUser); err != nil {
		log.Printf("handleStart Create: %v", err)
		return
	}

	h.states.SetState(userID, StateWaitStart)
	h.send(msg.Chat.ID,
		"Привет! 👋 Я помогу тебе отслеживать вредные привычки.\nНажми кнопку, чтобы начать.",
		startKeyboard(),
	)
}

func (h *Handler) handleRegistrationConfirm(msg *tgbotapi.Message) {
	h.states.SetState(msg.From.ID, StateIdle)
	h.send(msg.Chat.ID, "Добро пожаловать! 🎉\nСоздайте вашу первую привычку.", createFirstHabitKeyboard())
}

func (h *Handler) goMain(chatID int64, userID int64) {
	h.deleteMenuMessageIfSet(chatID, userID)
	h.states.SetState(userID, StateIdle)
	h.showMain(chatID, userID)
}

func (h *Handler) deleteMenuMessageIfSet(chatID int64, userID int64) {
	mid := h.states.GetMenuMessageID(userID)
	if mid != 0 {
		_, _ = h.bot.Request(tgbotapi.NewDeleteMessage(chatID, mid))
		h.states.SetMenuMessageID(userID, 0)
	}
}

// showMain: edit существующей главной из БД, иначе send.
func (h *Handler) showMain(chatID int64, userID int64) {
	habits, err := h.habitRepo.GetByUserID(userID)
	if err != nil {
		log.Printf("showMain GetByUserID: %v", err)
		return
	}

	if len(habits) == 0 {
		h.states.SetState(userID, StateIdle)
		h.send(chatID, "У вас нет привычек. Создайте первую!", createFirstHabitKeyboard())
		return
	}

	statsSlice := h.buildAllStats(habits)
	text := RenderMainScreen(habits, statsSlice)
	h.states.SetState(userID, StateIdle)
	inlineKb := mainInlineKeyboard(habits)

	_, mainID, ok, err := h.userRepo.GetMainMessage(userID)
	if err != nil {
		log.Printf("showMain GetMainMessage: %v", err)
	}
	if ok && mainID != 0 {
		if err := h.editHTML(chatID, mainID, text, inlineKb); err == nil {
			h.states.SetMainMessageID(userID, mainID)
			return
		} else if !isNotModified(err) {
			if isEditNotFound(err) {
				_ = h.userRepo.ClearMainMessage(userID)
			} else {
				log.Printf("showMain edit: %v", err)
			}
		} else {
			h.states.SetMainMessageID(userID, mainID)
			return
		}
	}

	sent, err := h.sendHTML(chatID, text, inlineKb)
	if err != nil {
		log.Printf("showMain send: %v", err)
		return
	}
	h.states.SetMainMessageID(userID, sent.MessageID)
	if err := h.userRepo.UpdateMainMessage(userID, chatID, sent.MessageID); err != nil {
		log.Printf("showMain UpdateMainMessage: %v", err)
	}
}

func (h *Handler) showMainMenuScreen(chatID int64, userID int64, editID int) {
	mid, err := h.editOrSend(chatID, editID, "Выберите действие:", mainMenuInlineKeyboard())
	if err != nil {
		log.Printf("showMainMenuScreen: %v", err)
		return
	}
	h.states.SetMenuMessageID(userID, mid)
}

func (h *Handler) showHabitMenu(chatID int64, userID int64, habitID int64, editID int) {
	h.states.SetState(userID, StateViewingHabitMenu)
	h.states.SetViewingHabitID(userID, habitID)
	mid, err := h.editOrSend(chatID, editID, "Выберите действие:", habitMenuInlineKeyboard(habitID))
	if err != nil {
		log.Printf("showHabitMenu: %v", err)
		return
	}
	h.states.SetMenuMessageID(userID, mid)
}

func (h *Handler) askConfirmRelapseByID(chatID int64, userID int64, habitID int64, back string, editID int) {
	habit, err := h.habitRepo.GetByID(habitID)
	if err != nil || habit == nil || habit.UserID != userID {
		h.goMain(chatID, userID)
		return
	}
	h.states.SetState(userID, StateWaitConfirmRelapse)
	h.states.SetPendingHabit(userID, habitID)
	if back == "menu" {
		h.states.SetReturnAfterRelapse(userID, habitID)
	} else {
		h.states.SetReturnAfterRelapse(userID, 0)
	}

	lastRelapseAt := habit.LastRelapseAt
	if lastRelapseAt.IsZero() {
		lastRelapseAt = habit.OriginAt
	}

	text := fmt.Sprintf(
		"Зарегистрировать срыв по привычке <b>%s</b>?\nПрошло %s с последнего срыва.",
		html.EscapeString(habit.Name),
		formatDuration(time.Since(lastRelapseAt)),
	)
	mid, err := h.editOrSend(chatID, editID, text, confirmRelapseInlineKeyboard(habitID, back))
	if err != nil {
		log.Printf("askConfirmRelapseByID: %v", err)
		return
	}
	h.states.SetMenuMessageID(userID, mid)
}

func (h *Handler) showHabitStatsByID(chatID int64, userID int64, habitID int64, editID int) {
	habit, err := h.habitRepo.GetByID(habitID)
	if err != nil || habit == nil || habit.UserID != userID {
		h.goMain(chatID, userID)
		return
	}

	st := h.calcHabitStats(*habit)
	last20, _ := h.relapseRepo.GetLast20ByHabitID(habit.ID)
	text := RenderStatsScreen(*habit, st, last20)

	h.states.SetState(userID, StateViewingHabitStats)
	h.states.SetViewingHabitID(userID, habitID)
	mid, err := h.editOrSend(chatID, editID, text, statsBackInlineKeyboard(habitID))
	if err != nil {
		log.Printf("showHabitStatsByID: %v", err)
		return
	}
	h.states.SetMenuMessageID(userID, mid)
}

func (h *Handler) declineRelapse(chatID int64, userID int64, habitID int64, back string, editID int) {
	h.states.SetReturnAfterRelapse(userID, 0)
	h.states.SetPendingHabit(userID, 0)
	h.states.SetState(userID, StateIdle)
	if back == "menu" && habitID != 0 {
		h.showHabitMenu(chatID, userID, habitID, editID)
		return
	}
	h.goMain(chatID, userID)
}

func (h *Handler) confirmRelapse(chatID int64, userID int64, habitID int64, callbackID string) {
	toast := "✅ Срыв зарегистрирован"
	if err := h.habitSvc.RegisterRelapse(habitID); err != nil {
		log.Printf("RegisterRelapse: %v", err)
		toast = "Ошибка при регистрации срыва"
		if _, cbErr := h.bot.Request(tgbotapi.NewCallback(callbackID, toast)); cbErr != nil {
			log.Printf("confirmRelapse callback: %v", cbErr)
		}
		return
	}
	if _, err := h.bot.Request(tgbotapi.NewCallback(callbackID, toast)); err != nil {
		log.Printf("confirmRelapse callback: %v", err)
	}
	h.states.SetReturnAfterRelapse(userID, 0)
	h.states.SetPendingHabit(userID, 0)
	h.deleteMenuMessageIfSet(chatID, userID)
	h.goMain(chatID, userID)
}

func (h *Handler) startHabitCreationByChat(chatID int64, userID int64) {
	_ = h.userRepo.ClearMainMessage(userID)
	h.deleteMenuMessageIfSet(chatID, userID)
	h.states.ResetDraft(userID)
	h.states.SetState(userID, StateHabitName)
	h.send(chatID,
		"📝 Создание новой привычки\n\nШаг 1/5: Введите название привычки или выберите из предложенных:",
		defaultHabitNamesKeyboard(),
	)
}

func (h *Handler) handleHabitCreationStep(msg *tgbotapi.Message, state State) {
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)
	draft := h.states.GetDraft(userID)

	switch state {
	case StateHabitName:
		if text == "" {
			h.send(msg.Chat.ID, "Название не может быть пустым. Попробуйте ещё раз:", defaultHabitNamesKeyboard())
			return
		}
		draft.Name = text
		h.states.SetState(userID, StateHabitLastRelapse)
		h.send(msg.Chat.ID,
			"Шаг 2/5: Введите дату и время последнего срыва (точка отсчёта):\nФормат: ДД.ММ.ГГГГ ЧЧ:ММ\nПример: 01.03.2026 09:00",
			removeKeyboard())

	case StateHabitLastRelapse:
		t, err := time.ParseInLocation(dtLayout, text, time.Local)
		if err != nil {
			h.send(msg.Chat.ID, "Неверный формат. Введите дату в формате ДД.ММ.ГГГГ ЧЧ:ММ\nПример: 01.03.2026 09:00", removeKeyboard())
			return
		}
		draft.OriginAt = t
		h.states.SetState(userID, StateHabitCost)
		h.send(msg.Chat.ID, "Шаг 3/5: Введите стоимость одного срыва (рублей):\nПример: 250", removeKeyboard())

	case StateHabitCost:
		cost, err := strconv.ParseFloat(text, 64)
		if err != nil || cost < 0 {
			h.send(msg.Chat.ID, "Введите корректное число (например: 250 или 99.50):", removeKeyboard())
			return
		}
		draft.CostPerRelapse = cost
		h.states.SetState(userID, StateHabitAvgPeriod)
		h.send(msg.Chat.ID, "Шаг 4/5: Выберите период для расчета среднего количества срывов:", periodKeyboard())

	case StateHabitAvgPeriod:
		period := parsePeriod(text)
		if period == "" {
			h.send(msg.Chat.ID, "Пожалуйста, выберите период с помощью кнопок:", periodKeyboard())
			return
		}
		draft.AvgRelapsesPeriod = period
		h.states.SetState(userID, StateHabitAvgCount)
		h.send(msg.Chat.ID, fmt.Sprintf("Шаг 5/5: Введите среднее количество срывов за %s:\nПример: 3", strings.ToLower(text)), removeKeyboard())

	case StateHabitAvgCount:
		count, err := strconv.ParseFloat(text, 64)
		if err != nil || count <= 0 {
			h.send(msg.Chat.ID, "Введите корректное число больше 0 (например: 2 или 0.5):", removeKeyboard())
			return
		}
		draft.AvgRelapsesCount = count

		originAt, ok := draft.OriginAt.(time.Time)
		if !ok {
			h.sendHTML(msg.Chat.ID, "Произошла ошибка. Начнём заново.", nil)
			h.startHabitCreationByChat(msg.Chat.ID, userID)
			return
		}

		svcDraft := service.HabitDraft{
			Name:              draft.Name,
			OriginAt:          originAt,
			CostPerRelapse:    draft.CostPerRelapse,
			AvgRelapsesCount:  draft.AvgRelapsesCount,
			AvgRelapsesPeriod: draft.AvgRelapsesPeriod,
		}

		_, err = h.habitSvc.CreateHabit(userID, svcDraft)
		if err != nil {
			log.Printf("CreateHabit: %v", err)
			h.sendHTML(msg.Chat.ID, "Ошибка при создании привычки. Попробуйте снова.", nil)
			return
		}

		h.states.SetState(userID, StateIdle)
		h.states.ResetDraft(userID)
		h.send(msg.Chat.ID, fmt.Sprintf("✅ Привычка %s создана!", draft.Name), removeKeyboard())
		h.goMain(msg.Chat.ID, userID)
	}
}

func (h *Handler) buildAllStats(habits []models.Habit) []service.HabitStats {
	stats := make([]service.HabitStats, len(habits))
	for i, habit := range habits {
		stats[i] = h.calcHabitStats(habit)
	}
	return stats
}

func (h *Handler) calcHabitStats(habit models.Habit) service.HabitStats {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := service.StatsLoadSince(habit, now)
	relapses, err := h.relapseRepo.GetByHabitIDSince(habit.ID, since)
	if err != nil {
		log.Printf("calcHabitStats GetByHabitIDSince: %v", err)
	}
	total, err := h.relapseRepo.CountByHabitID(habit.ID)
	if err != nil {
		log.Printf("calcHabitStats CountByHabitID: %v", err)
		total = len(relapses)
	}
	before, err := h.relapseRepo.CountByHabitIDBefore(habit.ID, startOfToday)
	if err != nil {
		log.Printf("calcHabitStats CountByHabitIDBefore: %v", err)
		before = 0
	}
	return h.statsSvc.CalcWithTotals(habit, relapses, total, before, now)
}

// editOrSend: edit overlay if possible, else send new. Returns message id.
func (h *Handler) editOrSend(chatID int64, editID int, text string, kb *tgbotapi.InlineKeyboardMarkup) (int, error) {
	if editID != 0 {
		if err := h.editHTML(chatID, editID, text, kb); err == nil || isNotModified(err) {
			return editID, nil
		} else if !isEditNotFound(err) {
			log.Printf("editOrSend edit: %v (fallback send)", err)
		}
	}
	sent, err := h.sendHTML(chatID, text, kb)
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

func (h *Handler) editHTML(chatID int64, msgID int, text string, kb *tgbotapi.InlineKeyboardMarkup) error {
	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		edit.ReplyMarkup = kb
	}
	_, err := h.bot.Send(edit)
	return err
}

func (h *Handler) sendHTML(chatID int64, text string, kb *tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableNotification = true
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	return h.bot.Send(msg)
}

// send — для wizard с ReplyKeyboard (без HTML-разметки в шаблонах).
func (h *Handler) send(chatID int64, text string, kb interface{}) tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableNotification = true
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	sent, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("send error: %v", err)
	}
	return sent
}

func isNotModified(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "message is not modified")
}

func isEditNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "message to edit not found")
}

func parsePeriod(text string) models.AvgPeriod {
	switch text {
	case "День":
		return models.PeriodDay
	case "Месяц":
		return models.PeriodMonth
	case "3 месяца":
		return models.Period3Month
	case "Полгода":
		return models.Period6Month
	case "Год":
		return models.PeriodYear
	}
	return ""
}
