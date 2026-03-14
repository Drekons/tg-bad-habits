package service

import (
	"fmt"
	"time"

	"github.com/drek/tg-bad-habbits/internal/models"
	"github.com/drek/tg-bad-habbits/internal/repository"
)

// HabitDraft is used during step-by-step habit creation.
type HabitDraft struct {
	Name              string
	OriginAt          time.Time
	CostPerRelapse    float64
	AvgRelapsesCount  float64
	AvgRelapsesPeriod models.AvgPeriod
}

// HabitService orchestrates habit-related business operations.
type HabitService struct {
	habitRepo   *repository.HabitRepo
	relapseRepo *repository.RelapseRepo
}

func NewHabitService(habitRepo *repository.HabitRepo, relapseRepo *repository.RelapseRepo) *HabitService {
	return &HabitService{
		habitRepo:   habitRepo,
		relapseRepo: relapseRepo,
	}
}

// CreateHabit creates a new habit from a draft.
func (s *HabitService) CreateHabit(userID int64, draft HabitDraft) (*models.Habit, error) {
	h := &models.Habit{
		UserID:            userID,
		Name:              draft.Name,
		OriginAt:          draft.OriginAt,
		LastRelapseAt:     draft.OriginAt, // initially equals origin
		CostPerRelapse:    draft.CostPerRelapse,
		AvgRelapsesCount:  draft.AvgRelapsesCount,
		AvgRelapsesPeriod: draft.AvgRelapsesPeriod,
	}

	id, err := s.habitRepo.Create(h)
	if err != nil {
		return nil, fmt.Errorf("HabitService.CreateHabit: %w", err)
	}
	h.ID = id
	return h, nil
}

// RegisterRelapse records a new relapse and updates last_relapse_at.
func (s *HabitService) RegisterRelapse(habitID int64) error {
	now := time.Now()

	relapse := &models.Relapse{
		HabitID:    habitID,
		RelapsedAt: now,
	}
	if err := s.relapseRepo.Create(relapse); err != nil {
		return fmt.Errorf("HabitService.RegisterRelapse Create: %w", err)
	}

	if err := s.habitRepo.UpdateLastRelapse(habitID, now); err != nil {
		return fmt.Errorf("HabitService.RegisterRelapse Update: %w", err)
	}

	return nil
}
