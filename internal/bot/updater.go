package bot

import (
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/drek/tg-bad-habbits/internal/repository"
	"github.com/drek/tg-bad-habbits/internal/service"
)

// Updater refreshes the main screen message for users who have it open, once per minute.
type Updater struct {
	bot         *tgbotapi.BotAPI
	userRepo    *repository.UserRepo
	habitRepo   *repository.HabitRepo
	relapseRepo *repository.RelapseRepo
	statsSvc    *service.StatsService
}

func NewUpdater(
	bot *tgbotapi.BotAPI,
	userRepo *repository.UserRepo,
	habitRepo *repository.HabitRepo,
	relapseRepo *repository.RelapseRepo,
	statsSvc *service.StatsService,
) *Updater {
	return &Updater{
		bot:         bot,
		userRepo:    userRepo,
		habitRepo:   habitRepo,
		relapseRepo: relapseRepo,
		statsSvc:    statsSvc,
	}
}

func (u *Updater) Start() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		u.refresh()
	}
}

func (u *Updater) refresh() {
	users, err := u.userRepo.GetUsersWithMainMessage()
	if err != nil {
		log.Printf("Updater GetUsersWithMainMessage: %v", err)
		return
	}
	if len(users) == 0 {
		return
	}
	log.Printf("Updater: refreshing %d user(s)", len(users))
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, m := range users {
		habits, err := u.habitRepo.GetByUserID(m.UserID)
		if err != nil || len(habits) == 0 {
			continue
		}

		statsSlice := make([]service.HabitStats, len(habits))
		for i, habit := range habits {
			since := service.StatsLoadSince(habit, now)
			relapses, err := u.relapseRepo.GetByHabitIDSince(habit.ID, since)
			if err != nil {
				continue
			}
			total, err := u.relapseRepo.CountByHabitID(habit.ID)
			if err != nil {
				total = len(relapses)
			}
			before, err := u.relapseRepo.CountByHabitIDBefore(habit.ID, startOfToday)
			if err != nil {
				before = 0
			}
			statsSlice[i] = u.statsSvc.CalcWithTotals(habit, relapses, total, before, now)
		}

		text := RenderMainScreen(habits, statsSlice)
		inlineKb := mainInlineKeyboard(habits)

		editMsg := tgbotapi.NewEditMessageText(m.ChatID, m.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeHTML
		editMsg.ReplyMarkup = inlineKb
		if _, err := u.bot.Send(editMsg); err != nil {
			errLow := strings.ToLower(err.Error())
			if strings.Contains(errLow, "message is not modified") {
				continue // already up to date — keep main_message_id
			}
			if strings.Contains(errLow, "message to edit not found") {
				_ = u.userRepo.ClearMainMessage(m.UserID)
			} else {
				log.Printf("Updater EditMessageText [user=%d]: %v", m.UserID, err)
			}
		}
	}
}
