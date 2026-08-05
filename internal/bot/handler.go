package bot

import (
	"fmt"
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

// Handle dispatches an incoming update.
func (h *Handler) Handle(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)

	// /start command is always handled first
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

	case StateWaitConfirmRelapse:
		switch text {
		case "✅ Да":
			h.handleRelapseConfirmed(msg)
		case "❌ Нет":
			h.handleRelapseDeclined(msg)
		default:
			h.send(msg.Chat.ID, "Пожалуйста, используйте кнопки ниже.", confirmRelapseKeyboard())
		}

	case StateHabitName, StateHabitLastRelapse, StateHabitCost, StateHabitAvgCount, StateHabitAvgPeriod:
		h.handleHabitCreationStep(msg, state)

	case StateViewingHabitMenu, StateViewingHabitStats:
		h.handleMainMenu(msg)

	default: // StateIdle
		h.handleMainMenu(msg)
	}
}

// HandleCallbackQuery handles inline button callbacks.
func (h *Handler) HandleCallbackQuery(cq *tgbotapi.CallbackQuery) {
	userID := cq.From.ID
	chatID := cq.Message.Chat.ID
	data := cq.Data

	// Answer callback so Telegram removes loading state
	if _, err := h.bot.Request(tgbotapi.NewCallback(cq.ID, "")); err != nil {
		log.Printf("HandleCallbackQuery Answer: %v", err)
	}

	switch {
	case data == "main_menu":
		h.showMainMenuScreen(chatID, userID)

	case data == "habit_back":
		h.returnFromHabitMenuToMain(chatID, userID)

	case strings.HasPrefix(data, "habit_menu:"):
		habitID, ok := h.parseOwnedHabitID(userID, strings.TrimPrefix(data, "habit_menu:"))
		if !ok {
			return
		}
		h.showHabitMenu(chatID, userID, habitID)

	case strings.HasPrefix(data, "habit_stats:"):
		habitID, ok := h.parseOwnedHabitID(userID, strings.TrimPrefix(data, "habit_stats:"))
		if !ok {
			return
		}
		h.showHabitStatsByID(chatID, userID, habitID)

	case strings.HasPrefix(data, "relapse:"):
		habitID, ok := h.parseOwnedHabitID(userID, strings.TrimPrefix(data, "relapse:"))
		if !ok {
			return
		}
		if h.states.GetState(userID) == StateViewingHabitMenu {
			h.states.SetReturnAfterRelapse(userID, habitID)
		} else {
			h.states.SetReturnAfterRelapse(userID, 0)
		}
		h.askConfirmRelapseByID(chatID, userID, habitID)
	}
}

// parseOwnedHabitID parses habit id and checks it belongs to userID.
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

// ─── /start ─────────────────────────────────────────────────────────────────

func (h *Handler) handleStart(msg *tgbotapi.Message) {
	userID := msg.From.ID
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("handleStart GetByID: %v", err)
		return
	}

	if user != nil {
		// Returning user
		habits, _ := h.habitRepo.GetByUserID(userID)
		h.states.SetState(userID, StateIdle)
		m := h.sendText(msg.Chat.ID, "С возвращением! 👋")
		_ = m
		if len(habits) > 0 {
			h.showMain(msg.Chat.ID, userID)
		} else {
			h.send(msg.Chat.ID, "У вас нет привычек. Создайте первую!", createFirstHabitKeyboard())
		}
		return
	}

	// New user — save and show start button
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
	userID := msg.From.ID
	h.states.SetState(userID, StateIdle)
	h.send(msg.Chat.ID, "Добро пожаловать! 🎉\nСоздайте вашу первую привычку.", createFirstHabitKeyboard())
}

// ─── Main menu ───────────────────────────────────────────────────────────────

func (h *Handler) handleMainMenu(msg *tgbotapi.Message) {
	userID := msg.From.ID
	text := strings.TrimSpace(msg.Text)
	state := h.states.GetState(userID)
	chatID := msg.Chat.ID

	// Меню привычки: Срыв / Статистика / Назад
	if state == StateViewingHabitMenu {
		habitID := h.states.GetViewingHabitID(userID)
		switch text {
		case "💥 Срыв":
			h.states.SetReturnAfterRelapse(userID, habitID)
			h.askConfirmRelapseByID(chatID, userID, habitID)
		case "📊 Статистика":
			h.showHabitStatsByID(chatID, userID, habitID)
		case "◀️ Назад", "Назад":
			h.returnFromHabitMenuToMain(chatID, userID)
		default:
			h.send(chatID, "Используйте кнопки ниже.", habitMenuReplyKeyboard())
		}
		return
	}

	// Статистика по привычке: Назад → меню привычки
	if state == StateViewingHabitStats {
		if text == "◀️ Назад" || strings.Contains(text, "Назад") {
			h.showHabitMenu(chatID, userID, h.states.GetViewingHabitID(userID))
		} else {
			h.showHabitMenu(chatID, userID, h.states.GetViewingHabitID(userID))
		}
		return
	}

	// StateIdle: главное меню (после callback «Меню» или с главного экрана)
	switch {
	case text == "➕ Добавить привычку" || text == "➕ Создать первую вредную привычку":
		h.startHabitCreation(msg)

	case text == "🏠 Перейти на главную":
		h.deleteMenuMessageIfSet(chatID, userID)
		h.showMain(chatID, userID)

	case text == "🏠 На основной экран" || strings.Contains(text, "На основной экран"):
		h.states.SetState(userID, StateIdle)
		h.deleteMenuMessageIfSet(chatID, userID)
		h.showMain(chatID, userID)

	default:
		h.deleteMenuMessageIfSet(chatID, userID)
		h.showMain(chatID, userID)
	}
}

func (h *Handler) returnFromHabitMenuToMain(chatID int64, userID int64) {
	mid := h.states.GetMenuMessageID(userID)
	if mid != 0 {
		_, _ = h.bot.Request(tgbotapi.NewDeleteMessage(chatID, mid))
		h.states.SetMenuMessageID(userID, 0)
	}
	oldMainID := h.states.GetMainMessageID(userID)
	if oldMainID != 0 {
		_, _ = h.bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMainID))
	}
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

// ─── Main screen ─────────────────────────────────────────────────────────────

// showMain показывает главный экран: одно сообщение с текстом и только inline-клавиатурой (без Reply).
// ID сохраняем для автообновления через EditMessageText/EditMessageReplyMarkup в Updater.
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
	sent, err := h.sendMarkdown(chatID, text, inlineKb)
	if err != nil {
		log.Printf("showMain send: %v", err)
		return
	}

	h.states.SetMainMessageID(userID, sent.MessageID)
	if err := h.userRepo.UpdateMainMessage(userID, chatID, sent.MessageID); err != nil {
		log.Printf("showMain UpdateMainMessage: %v", err)
	}
}

// showMainMenuScreen отправляет сообщение «Выберите действие» с Reply «Добавить привычку»/«Перейти на главную».
func (h *Handler) showMainMenuScreen(chatID int64, userID int64) {
	sent := h.send(chatID, "Выберите действие:", mainMenuReplyKeyboard())
	h.states.SetMenuMessageID(userID, sent.MessageID)
}

// showHabitMenu — inline-кнопки с habitID (статистика работает даже без in-memory state / при втором инстансе).
func (h *Handler) showHabitMenu(chatID int64, userID int64, habitID int64) {
	h.states.SetState(userID, StateViewingHabitMenu)
	h.states.SetViewingHabitID(userID, habitID)
	_ = h.userRepo.ClearMainMessage(userID)
	// Снять старую Reply-клавиатуру, затем меню на inline.
	h.send(chatID, "Выберите действие:", removeKeyboard())
	sent, err := h.sendMarkdown(chatID, "Выберите действие:", habitMenuInlineKeyboard(habitID))
	if err != nil {
		log.Printf("showHabitMenu: %v", err)
		return
	}
	h.states.SetMenuMessageID(userID, sent.MessageID)
}

// askConfirmRelapseByID показывает экран подтверждения срыва по habitID, задаёт ReturnAfterRelapse.
func (h *Handler) askConfirmRelapseByID(chatID int64, userID int64, habitID int64) {
	habit, err := h.habitRepo.GetByID(habitID)
	if err != nil || habit == nil || habit.UserID != userID {
		h.showMain(chatID, userID)
		return
	}
	_ = h.userRepo.ClearMainMessage(userID)
	h.states.SetState(userID, StateWaitConfirmRelapse)
	h.states.SetPendingHabit(userID, habitID)

	lastRelapseAt := habit.LastRelapseAt
	if lastRelapseAt.IsZero() {
		lastRelapseAt = habit.OriginAt
	}
	timeSinceLastRelapse := time.Since(lastRelapseAt)

	h.send(chatID,
		fmt.Sprintf(
			"Зарегистрировать срыв по привычке *%s*?\nПрошло %s с последнего срыва.",
			escapeMarkdown(habit.Name),
			formatDuration(timeSinceLastRelapse),
		),
		confirmRelapseKeyboard(),
	)
}

// ─── Stats screens ────────────────────────────────────────────────────────────

func (h *Handler) showHabitStatsByID(chatID int64, userID int64, habitID int64) {
	habit, err := h.habitRepo.GetByID(habitID)
	if err != nil || habit == nil || habit.UserID != userID {
		h.showMain(chatID, userID)
		return
	}

	st := h.calcHabitStats(*habit)
	last20, _ := h.relapseRepo.GetLast20ByHabitID(habit.ID)
	text := RenderStatsScreen(*habit, st, last20)

	h.states.SetState(userID, StateViewingHabitStats)
	h.states.SetViewingHabitID(userID, habitID)
	if _, err := h.sendMarkdown(chatID, text, backKeyboard()); err != nil {
		log.Printf("showHabitStatsByID send: %v", err)
	}
}

// ─── Relapse flow ─────────────────────────────────────────────────────────────

func (h *Handler) handleRelapseDeclined(msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	returnID := h.states.GetReturnAfterRelapse(userID)
	h.states.SetReturnAfterRelapse(userID, 0)
	h.states.SetState(userID, StateIdle)
	h.send(chatID, "Отменено.", removeKeyboard())
	if returnID != 0 {
		h.showHabitMenu(chatID, userID, returnID)
	} else {
		h.showMain(chatID, userID)
	}
}

func (h *Handler) handleRelapseConfirmed(msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	habitID := h.states.GetPendingHabit(userID)

	if err := h.habitSvc.RegisterRelapse(habitID); err != nil {
		log.Printf("RegisterRelapse: %v", err)
		h.send(chatID, "Ошибка при регистрации срыва. Попробуйте снова.", confirmRelapseKeyboard())
		return
	}

	h.states.SetReturnAfterRelapse(userID, 0)
	h.send(chatID, "✅ Срыв зарегистрирован.", removeKeyboard())
	h.showMain(chatID, userID) // по ТЗ после «Да» всегда на главную
}

// ─── Habit creation flow ──────────────────────────────────────────────────────

func (h *Handler) startHabitCreation(msg *tgbotapi.Message) {
	userID := msg.From.ID
	_ = h.userRepo.ClearMainMessage(userID) // автообновление только на главном экране
	h.states.ResetDraft(userID)
	h.states.SetState(userID, StateHabitName)
	h.send(msg.Chat.ID,
		"📝 Создание новой привычки\n\nШаг 1/5: Введите *название* привычки или выберите из предложенных:",
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
			"Шаг 2/5: Введите *дату и время последнего срыва* (точка отсчёта):\nФормат: ДД.ММ.ГГГГ ЧЧ:ММ\nПример: 01.03.2026 09:00",
			removeKeyboard())

	case StateHabitLastRelapse:
		t, err := time.ParseInLocation(dtLayout, text, time.Local)
		if err != nil {
			h.send(msg.Chat.ID, "Неверный формат. Введите дату в формате ДД.ММ.ГГГГ ЧЧ:ММ\nПример: 01.03.2026 09:00", removeKeyboard())
			return
		}
		draft.OriginAt = t
		h.states.SetState(userID, StateHabitCost)
		h.send(msg.Chat.ID, "Шаг 3/5: Введите *стоимость одного срыва* (рублей):\nПример: 250", removeKeyboard())

	case StateHabitCost:
		cost, err := strconv.ParseFloat(text, 64)
		if err != nil || cost < 0 {
			h.send(msg.Chat.ID, "Введите корректное число (например: 250 или 99.50):", removeKeyboard())
			return
		}
		draft.CostPerRelapse = cost
		h.states.SetState(userID, StateHabitAvgPeriod)
		h.send(msg.Chat.ID, "Шаг 4/5: Выберите *период* для расчета среднего количества срывов:", periodKeyboard())

	case StateHabitAvgPeriod:
		period := parsePeriod(text)
		if period == "" {
			h.send(msg.Chat.ID, "Пожалуйста, выберите период с помощью кнопок:", periodKeyboard())
			return
		}
		draft.AvgRelapsesPeriod = period

		h.states.SetState(userID, StateHabitAvgCount)
		h.send(msg.Chat.ID, fmt.Sprintf("Шаг 5/5: Введите *среднее количество срывов* за %s:\nПример: 3", strings.ToLower(text)), removeKeyboard())

	case StateHabitAvgCount:
		count, err := strconv.ParseFloat(text, 64)
		if err != nil || count <= 0 {
			h.send(msg.Chat.ID, "Введите корректное число больше 0 (например: 2 или 0.5):", removeKeyboard())
			return
		}
		draft.AvgRelapsesCount = count

		originAt, ok := draft.OriginAt.(time.Time)
		if !ok {
			h.sendText(msg.Chat.ID, "Произошла ошибка. Начнём заново.")
			h.startHabitCreation(msg)
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
			h.sendText(msg.Chat.ID, "Ошибка при создании привычки. Попробуйте снова.")
			return
		}

		h.states.SetState(userID, StateIdle)
		h.states.ResetDraft(userID)
		h.send(msg.Chat.ID, fmt.Sprintf("✅ Привычка *%s* создана!", escapeMarkdown(draft.Name)), removeKeyboard())
		h.showMain(msg.Chat.ID, userID) // сброс клавиатуры и главный экран с кнопками
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// Все сообщения бота отправляются без push-уведомлений (DisableNotification = true в send/sendText/sendMarkdown и в Updater).

func (h *Handler) buildAllStats(habits []models.Habit) []service.HabitStats {
	stats := make([]service.HabitStats, len(habits))
	for i, habit := range habits {
		stats[i] = h.calcHabitStats(habit)
	}
	return stats
}

// calcHabitStats loads a year window of relapses + lifetime count (avg per period).
func (h *Handler) calcHabitStats(habit models.Habit) service.HabitStats {
	now := time.Now()
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
	return h.statsSvc.CalcWithTotal(habit, relapses, total, now)
}

func (h *Handler) send(chatID int64, text string, kb interface{}) tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
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

func (h *Handler) sendText(chatID int64, text string) tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableNotification = true
	sent, _ := h.bot.Send(msg)
	return sent
}

func (h *Handler) sendMarkdown(chatID int64, text string, kb interface{}) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableNotification = true
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	return h.bot.Send(msg)
}

// parsePeriod maps button text to models.AvgPeriod.
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
