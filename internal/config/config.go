package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	BotToken        string
	DBDSN           string
	DBMigrationsPath string
}

// Load reads configuration from .env file and environment variables.
func Load() (*Config, error) {
	// .env is optional (e.g. on production env vars may already be set)
	_ = godotenv.Load()

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}

	migrationsPath := os.Getenv("DB_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "internal/migrations"
	}

	return &Config{
		BotToken:        token,
		DBDSN:           dsn,
		DBMigrationsPath: migrationsPath,
	}, nil
}
