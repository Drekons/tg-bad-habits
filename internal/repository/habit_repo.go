package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/drek/tg-bad-habbits/internal/models"
)

// HabitRepo handles database operations for habits.
type HabitRepo struct {
	db *sqlx.DB
}

func NewHabitRepo(db *sqlx.DB) *HabitRepo {
	return &HabitRepo{db: db}
}

// GetByID returns a habit by ID. Caller must verify habit.UserID == userID if needed.
func (r *HabitRepo) GetByID(habitID int64) (*models.Habit, error) {
	var h models.Habit
	err := r.db.Get(&h,
		`SELECT id, user_id, name, origin_at, last_relapse_at,
		        cost_per_relapse, avg_relapses_count, avg_relapses_period, created_at
		 FROM habits WHERE id = ?`,
		habitID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("HabitRepo.GetByID: %w", err)
	}
	return &h, nil
}

// GetByUserID returns all habits for a user.
func (r *HabitRepo) GetByUserID(userID int64) ([]models.Habit, error) {
	var habits []models.Habit
	err := r.db.Select(&habits,
		`SELECT id, user_id, name, origin_at, last_relapse_at,
		        cost_per_relapse, avg_relapses_count, avg_relapses_period, created_at
		 FROM habits WHERE user_id = ? ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("HabitRepo.GetByUserID: %w", err)
	}
	return habits, nil
}

// Create inserts a new habit and returns its generated ID.
func (r *HabitRepo) Create(h *models.Habit) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO habits (user_id, name, origin_at, last_relapse_at, cost_per_relapse,
		                     avg_relapses_count, avg_relapses_period, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`,
		h.UserID, h.Name, h.OriginAt, h.LastRelapseAt,
		h.CostPerRelapse, h.AvgRelapsesCount, h.AvgRelapsesPeriod,
	)
	if err != nil {
		return 0, fmt.Errorf("HabitRepo.Create: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// UpdateLastRelapse updates only last_relapse_at for the given habit.
func (r *HabitRepo) UpdateLastRelapse(habitID int64, t time.Time) error {
	_, err := r.db.Exec("UPDATE habits SET last_relapse_at = ? WHERE id = ?", t, habitID)
	if err != nil {
		return fmt.Errorf("HabitRepo.UpdateLastRelapse: %w", err)
	}
	return nil
}
