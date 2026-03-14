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
	if len(users) == 0 {
		log.Printf("Updater: 0 users with main_message_id in DB (автообновление не сработает до открытия главной без «На основной экран»)")
		return
	}
	log.Printf("Updater: refreshing %d user(s)", len(users))
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

		// Сообщение с Reply Keyboard нельзя редактировать — удаляем и отправляем заново. Request() вместо Send(),
		// т.к. Telegram возвращает для deleteMessage только true, а Send() пытается разобрать ответ как Message и падает с json unmarshal.
		if _, err := u.bot.Request(tgbotapi.NewDeleteMessage(m.ChatID, m.MessageID)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "message to delete not found") {
				_ = u.userRepo.ClearMainMessage(m.UserID)
			} else {
				log.Printf("Updater delete [user=%d]: %v", m.UserID, err)
			}
			continue
		}

		msg := tgbotapi.NewMessage(m.ChatID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = mainKeyboard(habits)
		msg.DisableNotification = true
		sent, err := u.bot.Send(msg)
		if err != nil {
			log.Printf("Updater send [user=%d]: %v", m.UserID, err)
			_ = u.userRepo.ClearMainMessage(m.UserID)
			continue
		}

		if err := u.userRepo.UpdateMainMessage(m.UserID, m.ChatID, sent.MessageID); err != nil {
			log.Printf("Updater UpdateMainMessage [user=%d]: %v", m.UserID, err)
		}
	}
}
