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
			h.showMain(msg.Chat.ID, userID)
		default:
			h.send(msg.Chat.ID, "Пожалуйста, используйте кнопки ниже.", confirmRelapseKeyboard())
		}

	case StateHabitName, StateHabitLastRelapse, StateHabitCost, StateHabitAvgCount, StateHabitAvgPeriod:
		h.handleHabitCreationStep(msg, state)

	default: // StateIdle
		h.handleMainMenu(msg)
	}
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

	// From single-habit stats screen only "Back" is valid; anything else returns to habit list
	if state == StateViewingHabitStats && text != "◀️ Назад" {
		h.states.SetState(userID, StateIdle)
		h.showStatsHabitList(msg)
		return
	}

	switch {
	case text == "➕ Добавить привычку" || text == "➕ Создать первую вредную привычку":
		h.startHabitCreation(msg)

	case text == "📊 Статистика":
		h.showStatsHabitList(msg)

	case text == "🏠 Перейти на главную":
		h.showMain(msg.Chat.ID, userID)

	case text == "◀️ Назад":
		if state == StateViewingHabitStats {
			h.states.SetState(userID, StateIdle)
			h.showStatsHabitList(msg)
		} else {
			h.showMain(msg.Chat.ID, userID)
		}

	case strings.HasPrefix(text, "📊 "):
		// Select specific habit for stats
		habitName := strings.TrimPrefix(text, "📊 ")
		h.showHabitStats(msg, habitName)

	default:
		// Check if it's a habit name button for relapse
		habits, _ := h.habitRepo.GetByUserID(userID)
		for _, habit := range habits {
			if text == habit.Name {
				h.askConfirmRelapse(msg, habit.Name)
				return
			}
		}
		h.showMain(msg.Chat.ID, userID)
	}
}

// ─── Main screen ─────────────────────────────────────────────────────────────

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

	// Send a separate message to attach the reply keyboard,
	// because Telegram API forbids editing messages that have a ReplyKeyboardMarkup.
	// Use zero-width space so the user does not see an extra line of text.
	kbMsg := tgbotapi.NewMessage(chatID, "\u200B")
	kbMsg.ReplyMarkup = mainKeyboard(habits)
	kbMsg.DisableNotification = true
	h.bot.Send(kbMsg)

	// Send the actual stats message without markup so it can be edited by Updater
	sent, err := h.sendMarkdown(chatID, text, nil)
	if err != nil {
		log.Printf("showMain send: %v", err)
		return
	}

	h.states.SetState(userID, StateIdle)
	h.states.SetMainMessageID(userID, sent.MessageID)
	if err := h.userRepo.UpdateMainMessage(userID, chatID, sent.MessageID); err != nil {
		log.Printf("showMain UpdateMainMessage: %v", err)
	}
}

// ─── Stats screens ────────────────────────────────────────────────────────────

func (h *Handler) showStatsHabitList(msg *tgbotapi.Message) {
	userID := msg.From.ID
	habits, err := h.habitRepo.GetByUserID(userID)
	if err != nil || len(habits) == 0 {
		h.send(msg.Chat.ID, "Нет привычек для отображения.", backKeyboard())
		return
	}
	h.send(msg.Chat.ID, "Выберите привычку:", statsHabitKeyboard(habits))
}

func (h *Handler) showHabitStats(msg *tgbotapi.Message, habitName string) {
	userID := msg.From.ID
	habits, _ := h.habitRepo.GetByUserID(userID)

	var target *models.Habit
	for i := range habits {
		if habits[i].Name == habitName {
			target = &habits[i]
			break
		}
	}
	if target == nil {
		h.send(msg.Chat.ID, "Привычка не найдена.", removeKeyboard())
		h.showMain(msg.Chat.ID, userID)
		return
	}

	relapses, _ := h.relapseRepo.GetByHabitID(target.ID)
	last20, _ := h.relapseRepo.GetLast20ByHabitID(target.ID)
	st := h.statsSvc.Calc(*target, relapses, time.Now())
	text := RenderStatsScreen(*target, st, last20)

	h.states.SetState(userID, StateViewingHabitStats)
	_, err := h.sendMarkdown(msg.Chat.ID, text, backKeyboard())
	if err != nil {
		log.Printf("showHabitStats: %v", err)
	}
}

// ─── Relapse flow ─────────────────────────────────────────────────────────────

func (h *Handler) askConfirmRelapse(msg *tgbotapi.Message, habitName string) {
	userID := msg.From.ID
	habits, _ := h.habitRepo.GetByUserID(userID)

	var target *models.Habit
	for i := range habits {
		if habits[i].Name == habitName {
			target = &habits[i]
			break
		}
	}
	if target == nil {
		h.showMain(msg.Chat.ID, userID)
		return
	}

	h.states.SetState(userID, StateWaitConfirmRelapse)
	h.states.SetPendingHabit(userID, target.ID)
	h.send(msg.Chat.ID,
		fmt.Sprintf("Зарегистрировать срыв по привычке *%s*?", escapeMarkdown(target.Name)),
		confirmRelapseKeyboard(),
	)
}

func (h *Handler) handleRelapseConfirmed(msg *tgbotapi.Message) {
	userID := msg.From.ID
	habitID := h.states.GetPendingHabit(userID)

	if err := h.habitSvc.RegisterRelapse(habitID); err != nil {
		log.Printf("RegisterRelapse: %v", err)
		h.send(msg.Chat.ID, "Ошибка при регистрации срыва. Попробуйте снова.", confirmRelapseKeyboard())
		return
	}

	h.send(msg.Chat.ID, "✅ Срыв зарегистрирован.", nil)
	h.showMain(msg.Chat.ID, userID)
}

// ─── Habit creation flow ──────────────────────────────────────────────────────

func (h *Handler) startHabitCreation(msg *tgbotapi.Message) {
	userID := msg.From.ID
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
		h.showMain(msg.Chat.ID, userID)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) buildAllStats(habits []models.Habit) []service.HabitStats {
	stats := make([]service.HabitStats, len(habits))
	for i, habit := range habits {
		relapses, _ := h.relapseRepo.GetByHabitID(habit.ID)
		stats[i] = h.statsSvc.Calc(habit, relapses, time.Now())
	}
	return stats
}

func (h *Handler) send(chatID int64, text string, kb interface{}) tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
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
	sent, _ := h.bot.Send(msg)
	return sent
}

func (h *Handler) sendMarkdown(chatID int64, text string, kb interface{}) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
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
