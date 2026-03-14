package models

import "time"

// AvgPeriod defines the period over which average relapses are counted.
type AvgPeriod string

const (
	PeriodDay     AvgPeriod = "day"
	PeriodMonth   AvgPeriod = "month"
	Period3Month  AvgPeriod = "3month"
	Period6Month  AvgPeriod = "6month"
	PeriodYear    AvgPeriod = "year"
)

// PeriodDays returns the number of days corresponding to the period.
func (p AvgPeriod) Days() float64 {
	switch p {
	case PeriodDay:
		return 1
	case PeriodMonth:
		return 30
	case Period3Month:
		return 90
	case Period6Month:
		return 180
	case PeriodYear:
		return 365
	default:
		return 1
	}
}

// PeriodLabel returns a human-readable Russian label for the period.
func (p AvgPeriod) Label() string {
	switch p {
	case PeriodDay:
		return "день"
	case PeriodMonth:
		return "месяц"
	case Period3Month:
		return "3 месяца"
	case Period6Month:
		return "полгода"
	case PeriodYear:
		return "год"
	default:
		return string(p)
	}
}

// Habit represents a bad habit being tracked.
type Habit struct {
	ID                int64     `db:"id"`
	UserID            int64     `db:"user_id"`
	Name              string    `db:"name"`
	OriginAt          time.Time `db:"origin_at"`          // Fixed reference point, never changes
	LastRelapseAt     time.Time `db:"last_relapse_at"`    // Updated on every relapse
	CostPerRelapse    float64   `db:"cost_per_relapse"`
	AvgRelapsesCount  float64   `db:"avg_relapses_count"`
	AvgRelapsesPeriod AvgPeriod `db:"avg_relapses_period"`
	CreatedAt         time.Time `db:"created_at"`
}
