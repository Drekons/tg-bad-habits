package main

import (
	"log"

	"github.com/drek/tg-bad-habbits/internal/bot"
	"github.com/drek/tg-bad-habbits/internal/config"
	"github.com/drek/tg-bad-habbits/internal/db"
	"github.com/drek/tg-bad-habbits/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Connect(cfg.DBDSN, cfg.DBMigrationsPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	userRepo := repository.NewUserRepo(database)
	habitRepo := repository.NewHabitRepo(database)
	relapseRepo := repository.NewRelapseRepo(database)

	b, err := bot.New(cfg.BotToken, userRepo, habitRepo, relapseRepo)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}

	b.Run()
}
