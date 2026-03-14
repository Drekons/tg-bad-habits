package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/drek/tg-bad-habbits/internal/models"
)

// UserRepo handles database operations for users.
type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

// GetByID returns a user by Telegram ID. Returns nil, nil if not found.
func (r *UserRepo) GetByID(telegramID int64) (*models.User, error) {
	var user models.User
	err := r.db.Get(&user, "SELECT id, username, created_at FROM users WHERE id = ?", telegramID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("UserRepo.GetByID: %w", err)
	}
	return &user, nil
}

// Create inserts a new user record.
func (r *UserRepo) Create(user *models.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (id, username, created_at) VALUES (?, ?, NOW())",
		user.ID, user.Username,
	)
	if err != nil {
		return fmt.Errorf("UserRepo.Create: %w", err)
	}
	return nil
}

// MainMessage identifies the main screen message for a user (for auto-update after redeploy).
type MainMessage struct {
	UserID    int64
	ChatID    int64
	MessageID int
}

// UpdateMainMessage saves the main screen message id so the updater can refresh it after redeploy.
func (r *UserRepo) UpdateMainMessage(userID, chatID int64, messageID int) error {
	_, err := r.db.Exec(
		"UPDATE users SET main_chat_id = ?, main_message_id = ? WHERE id = ?",
		chatID, messageID, userID,
	)
	if err != nil {
		return fmt.Errorf("UserRepo.UpdateMainMessage: %w", err)
	}
	return nil
}

// ClearMainMessage clears the stored main message (e.g. when edit fails).
func (r *UserRepo) ClearMainMessage(userID int64) error {
	_, err := r.db.Exec(
		"UPDATE users SET main_chat_id = NULL, main_message_id = NULL WHERE id = ?",
		userID,
	)
	if err != nil {
		return fmt.Errorf("UserRepo.ClearMainMessage: %w", err)
	}
	return nil
}

// GetUsersWithMainMessage returns all users that have a main screen message to refresh.
func (r *UserRepo) GetUsersWithMainMessage() ([]MainMessage, error) {
	var rows []struct {
		ID            int64 `db:"id"`
		MainChatID    int64 `db:"main_chat_id"`
		MainMessageID int   `db:"main_message_id"`
	}
	err := r.db.Select(&rows,
		"SELECT id, main_chat_id, main_message_id FROM users WHERE main_chat_id IS NOT NULL AND main_message_id IS NOT NULL")
	if err != nil {
		return nil, fmt.Errorf("UserRepo.GetUsersWithMainMessage: %w", err)
	}
	out := make([]MainMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, MainMessage{UserID: row.ID, ChatID: row.MainChatID, MessageID: row.MainMessageID})
	}
	return out, nil
}
