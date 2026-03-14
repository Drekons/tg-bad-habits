package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/drek/tg-bad-habbits/internal/models"
)

// RelapseRepo handles database operations for relapses.
type RelapseRepo struct {
	db *sqlx.DB
}

func NewRelapseRepo(db *sqlx.DB) *RelapseRepo {
	return &RelapseRepo{db: db}
}

// Create inserts a new relapse event.
func (r *RelapseRepo) Create(relapse *models.Relapse) error {
	_, err := r.db.Exec(
		"INSERT INTO relapses (habit_id, relapsed_at) VALUES (?, ?)",
		relapse.HabitID, relapse.RelapsedAt,
	)
	if err != nil {
		return fmt.Errorf("RelapseRepo.Create: %w", err)
	}
	return nil
}

// GetByHabitID returns all relapses for a habit ordered by time ascending.
func (r *RelapseRepo) GetByHabitID(habitID int64) ([]models.Relapse, error) {
	var relapses []models.Relapse
	err := r.db.Select(&relapses,
		"SELECT id, habit_id, relapsed_at FROM relapses WHERE habit_id = ? ORDER BY relapsed_at ASC",
		habitID,
	)
	if err != nil {
		return nil, fmt.Errorf("RelapseRepo.GetByHabitID: %w", err)
	}
	return relapses, nil
}

// GetLast20ByHabitID returns the last 20 relapses, ordered newest first.
func (r *RelapseRepo) GetLast20ByHabitID(habitID int64) ([]models.Relapse, error) {
	var relapses []models.Relapse
	err := r.db.Select(&relapses,
		"SELECT id, habit_id, relapsed_at FROM relapses WHERE habit_id = ? ORDER BY relapsed_at DESC LIMIT 20",
		habitID,
	)
	if err != nil {
		return nil, fmt.Errorf("RelapseRepo.GetLast20ByHabitID: %w", err)
	}
	return relapses, nil
}

// GetByHabitIDUntil returns all relapses of a habit up to (and including) the given time.
func (r *RelapseRepo) GetByHabitIDUntil(habitID int64, until time.Time) ([]models.Relapse, error) {
	var relapses []models.Relapse
	err := r.db.Select(&relapses,
		"SELECT id, habit_id, relapsed_at FROM relapses WHERE habit_id = ? AND relapsed_at <= ? ORDER BY relapsed_at ASC",
		habitID, until,
	)
	if err != nil {
		return nil, fmt.Errorf("RelapseRepo.GetByHabitIDUntil: %w", err)
	}
	return relapses, nil
}
