package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/drek/tg-bad-habbits/internal/repository"
	"github.com/drek/tg-bad-habbits/internal/service"
)

// Bot ties together the Telegram API, handler, and updater.
type Bot struct {
	api     *tgbotapi.BotAPI
	handler *Handler
	updater *Updater
}

// New creates and wires up a Bot instance.
func New(
	token string,
	userRepo *repository.UserRepo,
	habitRepo *repository.HabitRepo,
	relapseRepo *repository.RelapseRepo,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	api.Debug = false
	log.Printf("Authorized as @%s", api.Self.UserName)

	states := NewStateManager()
	statsSvc := service.NewStatsService()
	habitSvc := service.NewHabitService(habitRepo, relapseRepo)

	handler := NewHandler(api, states, userRepo, habitRepo, relapseRepo, habitSvc, statsSvc)
	updater := NewUpdater(api, userRepo, habitRepo, relapseRepo, statsSvc)

	return &Bot{
		api:     api,
		handler: handler,
		updater: updater,
	}, nil
}

// Run starts the polling loop and background updater.
func (b *Bot) Run() {
	go b.updater.Start()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	log.Println("Bot is running...")

	for update := range updates {
		if update.CallbackQuery != nil {
			go b.handler.HandleCallbackQuery(update.CallbackQuery)
			continue
		}
		if update.Message != nil {
			go b.handler.Handle(update)
		}
	}
}
