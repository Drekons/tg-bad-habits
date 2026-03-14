package bot

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/drek/tg-bad-habbits/internal/repository"
	"github.com/drek/tg-bad-habbits/internal/service"
)

// Updater refreshes the main screen message for users who have it open, once per minute.
// It uses the DB-stored main_message_id so refresh keeps working after app redeploy.
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

// Start launches the background ticker. Should be called in a goroutine.
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
	now := time.Now()
	for _, m := range users {
		habits, err := u.habitRepo.GetByUserID(m.UserID)
		if err != nil || len(habits) == 0 {
			continue
		}

		statsSlice := make([]service.HabitStats, len(habits))
		for i, habit := range habits {
			relapses, err := u.relapseRepo.GetByHabitID(habit.ID)
			if err != nil {
				continue
			}
			statsSlice[i] = u.statsSvc.Calc(habit, relapses, now)
		}

		text := RenderMainScreen(habits, statsSlice)

		edit := tgbotapi.NewEditMessageText(m.ChatID, m.MessageID, text)
		edit.ParseMode = tgbotapi.ModeMarkdown

		if _, err := u.bot.Send(edit); err != nil {
			log.Printf("Updater edit [user=%d]: %v", m.UserID, err)
			_ = u.userRepo.ClearMainMessage(m.UserID)
		}
	}
}
