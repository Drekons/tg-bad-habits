package models

import "time"

// Relapse represents a single registered relapse event.
type Relapse struct {
	ID         int64     `db:"id"`
	HabitID    int64     `db:"habit_id"`
	RelapsedAt time.Time `db:"relapsed_at"`
}
