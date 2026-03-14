package service

import (
	"testing"
	"time"

	"github.com/drek/tg-bad-habbits/internal/models"
)

func habitBase() models.Habit {
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return models.Habit{
		ID:                1,
		UserID:            100,
		Name:              "Test",
		OriginAt:          origin,
		LastRelapseAt:     origin,
		CostPerRelapse:    100,
		AvgRelapsesCount:  2,
		AvgRelapsesPeriod: models.PeriodDay, // 2 relapses per day "budget"
	}
}

// ─── calcBalance ─────────────────────────────────────────────────────────────

func TestCalcBalance_NoRelapses(t *testing.T) {
	h := habitBase()
	// 10 days later, 0 real relapses. PeriodDay: effective days = (240-8*9)/16 = 10.5
	until := h.OriginAt.Add(10 * 24 * time.Hour)
	// potentialLoss = 2/day * 10.5 effective days * 100₽ = 2100, realLoss = 0, balance = +2100
	got := calcBalance(h, nil, until)
	if got != 2100 {
		t.Errorf("expected 2100, got %v", got)
	}
}

func TestCalcBalance_WithRelapses(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(10 * 24 * time.Hour)

	relapses := make([]models.Relapse, 15)
	for i := range relapses {
		relapses[i] = models.Relapse{HabitID: 1, RelapsedAt: h.OriginAt.Add(time.Duration(i) * 12 * time.Hour)}
	}

	// potentialLoss = 2 * 10.5 effective days * 100 = 2100, realLoss = 1500, balance = +600
	got := calcBalance(h, relapses, until)
	if got != 600 {
		t.Errorf("expected 600, got %v", got)
	}
}

func TestCalcBalance_OverBudget(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(10 * 24 * time.Hour)

	relapses := make([]models.Relapse, 25)
	for i := range relapses {
		relapses[i] = models.Relapse{HabitID: 1, RelapsedAt: h.OriginAt.Add(time.Duration(i) * 8 * time.Hour)}
	}

	// potentialLoss = 2100 (10.5 eff days), realLoss = 2500 → balance = -400
	got := calcBalance(h, relapses, until)
	if got != -400 {
		t.Errorf("expected -400, got %v", got)
	}
}

func TestCalcBalance_UntilBeforeOrigin(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(-1 * time.Hour)
	got := calcBalance(h, nil, until)
	if got != 0 {
		t.Errorf("expected 0 when until < origin, got %v", got)
	}
}

// Intermediate balance: habit with period month, no relapses, 15 days passed.
func TestCalcBalance_IntermediateBalance_MonthPeriod(t *testing.T) {
	h := habitBase()
	h.AvgRelapsesPeriod = models.PeriodMonth
	h.AvgRelapsesCount = 1
	until := h.OriginAt.Add(15 * 24 * time.Hour) // 15 days
	got := calcBalance(h, nil, until)
	// potential = (1/30)*15*cost = 0.5*100 = 50, real = 0
	if got != 50 {
		t.Errorf("expected intermediate balance 50, got %v", got)
	}
}

// Registration day: first 24h do not subtract sleep (daily habit).
func TestCalcAvgTimeBetween_RegistrationDay_NoSleepSubtract(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(24 * time.Hour) // exactly 1 day
	// fullDays=1 → sleeping=0, waking=24h, intervals=1 → avg=24h
	got := calcAvgTimeBetween(h, nil, until)
	if got != 24*time.Hour {
		t.Errorf("expected 24h on registration day (no sleep subtract), got %v", got)
	}
}

// ─── calcAvgTimeBetween ───────────────────────────────────────────────────────

func TestCalcAvgTimeBetween_NoRelapses(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(24 * time.Hour)
	// 1 day passed, 0 relapses. intervals = 1. avg = 24h
	got := calcAvgTimeBetween(h, nil, until)
	if got != 24*time.Hour {
		t.Errorf("expected 24h for empty relapses after 1 day, got %v", got)
	}
}

func TestCalcAvgTimeBetween_OneRelapse(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(24 * time.Hour) // 1 day
	relapses := []models.Relapse{{RelapsedAt: h.OriginAt.Add(12 * time.Hour)}}
	// intervals = 2. Total time = 24h. avg = 12h
	got := calcAvgTimeBetween(h, relapses, until)
	expected := 12 * time.Hour
	if got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestCalcAvgTimeBetween_MultipleRelapses(t *testing.T) {
	h := habitBase()
	until := h.OriginAt.Add(4 * 24 * time.Hour) // 4 days elapsed
	relapses := []models.Relapse{
		{RelapsedAt: h.OriginAt.Add(24 * time.Hour)}, // day 1
		{RelapsedAt: h.OriginAt.Add(48 * time.Hour)}, // day 2
		{RelapsedAt: h.OriginAt.Add(72 * time.Hour)}, // day 3
	} // 3 relapses. intervals = 4. PeriodDay: waking = 96-8*3 = 72h, avg = 72/4 = 18h
	got := calcAvgTimeBetween(h, relapses, until)
	expected := 18 * time.Hour
	if got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// ─── PeriodDays ───────────────────────────────────────────────────────────────

func TestPeriodDays(t *testing.T) {
	cases := []struct {
		period   models.AvgPeriod
		expected float64
	}{
		{models.PeriodDay, 1},
		{models.PeriodMonth, 30},
		{models.Period3Month, 90},
		{models.Period6Month, 180},
		{models.PeriodYear, 365},
	}
	for _, c := range cases {
		got := c.period.Days()
		if got != c.expected {
			t.Errorf("period %s: expected %v days, got %v", c.period, c.expected, got)
		}
	}
}

// ─── filterUntil ─────────────────────────────────────────────────────────────

func TestFilterUntil(t *testing.T) {
	base := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	relapses := []models.Relapse{
		{RelapsedAt: base.Add(-24 * time.Hour)}, // before
		{RelapsedAt: base.Add(-1 * time.Hour)},  // before
		{RelapsedAt: base},                       // exactly at cutoff — should be excluded (strict <)
		{RelapsedAt: base.Add(1 * time.Hour)},   // after
	}
	got := filterUntil(relapses, base)
	if len(got) != 2 {
		t.Errorf("expected 2 items before cutoff, got %d", len(got))
	}
}

// ─── StatsService.Calc ───────────────────────────────────────────────────────

func TestStatsService_Calc_ZeroRelapses(t *testing.T) {
	s := NewStatsService()
	h := habitBase()
	now := h.OriginAt.Add(48 * time.Hour) // 2 days passed

	// PeriodDay: effective days = (48-8)/16 = 2.5. Potential = 2*2.5*100 = 500. Balance = +500.
	// Avg time: waking 40h, intervals 1 → 40h.
	// Avg per period: 0/2 = 0.

	got := s.Calc(h, nil, now)

	if got.Balance != 500 {
		t.Errorf("expected balance 500, got %v", got.Balance)
	}
	if got.AvgTimeBetween != 40*time.Hour {
		t.Errorf("expected avg time 40h, got %v", got.AvgTimeBetween)
	}
	if got.AvgPerPeriod != 0 {
		t.Errorf("expected avg per period 0, got %v", got.AvgPerPeriod)
	}
}

