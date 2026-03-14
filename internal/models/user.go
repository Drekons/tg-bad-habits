package models

import "time"

// User represents a registered Telegram user.
type User struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	CreatedAt time.Time `db:"created_at"`
}
