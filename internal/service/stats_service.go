package service

import (
	"math"
	"time"

	"github.com/drek/tg-bad-habbits/internal/models"
)

// TrendData holds a computed metric with its delta vs "yesterday".
type TrendData struct {
	Current  float64
	Previous float64 // computed up to start of today
	Delta    float64
	Up       bool // true if increasing (worse for time, better for balance)
}

// HabitStats holds all calculated statistics for a single habit.
type HabitStats struct {
	Balance           float64
	BalanceTrend      TrendData
	AvgTimeBetween    time.Duration
	AvgTimeTrend      TrendData
	AvgPerPeriod      float64 // real average relapses per habit's period
	AvgPerPeriodTrend TrendData
	RelapsesInPeriod  int // count of relapses in current period (day/month/...) for habit
}

// StatsService computes all statistics from raw data.
type StatsService struct{}

func NewStatsService() *StatsService {
	return &StatsService{}
}

// Calc returns full HabitStats for a habit.
// relapses may be a windowed slice (e.g. last year); totalRelapses is the lifetime count for AvgPerPeriod.
// If totalRelapses < 0, len(relapses) is used.
func (s *StatsService) Calc(habit models.Habit, relapses []models.Relapse, now time.Time) HabitStats {
	return s.CalcWithTotal(habit, relapses, -1, now)
}

// CalcWithTotal is like Calc but uses totalRelapses for the lifetime average (when relapses is windowed).
func (s *StatsService) CalcWithTotal(habit models.Habit, relapses []models.Relapse, totalRelapses int, now time.Time) HabitStats {
	if totalRelapses < 0 {
		totalRelapses = len(relapses)
	}

	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	relapsesUntilYesterday := filterUntil(relapses, startOfToday)
	totalUntilYesterday := countUntil(totalRelapses, relapses, startOfToday)

	balanceNow := calcBalance(habit, relapses, now)
	balanceYesterday := calcBalance(habit, relapsesUntilYesterday, startOfToday)

	avgTimeNow := calcAvgTimeBetween(habit, relapses, now)
	avgTimeYesterday := calcAvgTimeBetween(habit, relapsesUntilYesterday, startOfToday)

	balanceDelta := balanceNow - balanceYesterday
	avgTimeDelta := avgTimeNow - avgTimeYesterday
	if totalRelapses == 0 {
		avgTimeDelta = 0
	}

	avgPerNow := calcAvgPerPeriod(habit, totalRelapses, now)
	avgPerYesterday := calcAvgPerPeriod(habit, totalUntilYesterday, startOfToday)
	avgPerDelta := avgPerNow - avgPerYesterday

	return HabitStats{
		Balance: balanceNow,
		BalanceTrend: TrendData{
			Current:  balanceNow,
			Previous: balanceYesterday,
			Delta:    balanceDelta,
			Up:       balanceDelta > 0,
		},
		AvgTimeBetween: avgTimeNow,
		AvgTimeTrend: TrendData{
			Current:  avgTimeNow.Hours(),
			Previous: avgTimeYesterday.Hours(),
			Delta:    avgTimeDelta.Hours(),
			Up:       avgTimeDelta > 0,
		},
		AvgPerPeriod: avgPerNow,
		AvgPerPeriodTrend: TrendData{
			Current:  avgPerNow,
			Previous: avgPerYesterday,
			Delta:    avgPerDelta,
			Up:       avgPerDelta > 0,
		},
		RelapsesInPeriod: countRelapsesInPeriod(habit, relapses, now),
	}
}

// countUntil estimates lifetime count as of cutoff when we only have a windowed slice + total.
// ponytail: if window doesn't reach origin, yesterday's total ≈ total - (relapses in window after cutoff); good enough for daily trend.
func countUntil(total int, window []models.Relapse, cutoff time.Time) int {
	after := 0
	for _, r := range window {
		if !r.RelapsedAt.Before(cutoff) {
			after++
		}
	}
	n := total - after
	if n < 0 {
		return 0
	}
	return n
}

// countRelapsesInPeriod returns the number of relapses in the current period for the habit.
func countRelapsesInPeriod(habit models.Habit, relapses []models.Relapse, now time.Time) int {
	start := periodStart(habit.AvgRelapsesPeriod, now)
	var n int
	for _, r := range relapses {
		if !r.RelapsedAt.Before(start) && !r.RelapsedAt.After(now) {
			n++
		}
	}
	return n
}

func periodStart(period models.AvgPeriod, now time.Time) time.Time {
	loc := now.Location()
	switch period {
	case models.PeriodDay:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	case models.PeriodMonth:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	case models.Period3Month, models.Period6Month, models.PeriodYear:
		days := int(period.Days())
		return now.AddDate(0, 0, -days)
	default:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
}

// balanceWindowStart: баланс всегда за последний год (или с origin, если привычка моложе).
func balanceWindowStart(habit models.Habit, until time.Time) time.Time {
	start := until.AddDate(0, 0, -365)
	if habit.OriginAt.After(start) {
		return habit.OriginAt
	}
	return start
}

// avgTimeWindowStart: для ежедневных — месяц, иначе — год.
func avgTimeWindowStart(habit models.Habit, until time.Time) time.Time {
	var start time.Time
	if habit.AvgRelapsesPeriod == models.PeriodDay {
		start = until.AddDate(0, 0, -30)
	} else {
		start = until.AddDate(0, 0, -365)
	}
	if habit.OriginAt.After(start) {
		return habit.OriginAt
	}
	return start
}

// StatsLoadSince returns the earliest time we need relapses for balance + avg time windows.
func StatsLoadSince(habit models.Habit, now time.Time) time.Time {
	b := balanceWindowStart(habit, now)
	a := avgTimeWindowStart(habit, now)
	if a.Before(b) {
		return a
	}
	return b
}

// calcBalance: potential − real за последний год (окно).
func calcBalance(habit models.Habit, relapses []models.Relapse, until time.Time) float64 {
	if until.Before(habit.OriginAt) {
		return 0
	}
	start := balanceWindowStart(habit, until)
	inWindow := filterSince(relapses, start, until)

	elapsed := until.Sub(start)
	if elapsed > 0 && elapsed < time.Minute {
		elapsed = time.Minute
	}
	var daysSince float64
	if habit.AvgRelapsesPeriod == models.PeriodDay {
		daysSince = effectiveWakingHours(elapsed) / 16
	} else {
		daysSince = elapsed.Hours() / 24
	}
	avgPerDay := habit.AvgRelapsesCount / habit.AvgRelapsesPeriod.Days()
	potentialLoss := avgPerDay * daysSince * habit.CostPerRelapse
	realLoss := float64(len(inWindow)) * habit.CostPerRelapse
	balance := math.Round((potentialLoss-realLoss)*100) / 100
	if len(inWindow) == 0 && until.After(habit.OriginAt) && balance <= 0 {
		daysSince = math.Max(daysSince, 1)
		avgPerDayMin := habit.AvgRelapsesCount / habit.AvgRelapsesPeriod.Days()
		if avgPerDayMin <= 0 {
			avgPerDayMin = 1.0 / 365
		}
		if habit.CostPerRelapse > 0 {
			potentialLoss = avgPerDayMin * daysSince * habit.CostPerRelapse
			return math.Round(potentialLoss*100) / 100
		}
	}
	return balance
}

// effectiveWakingHours: первые 24ч полностью; дальше 16/24 на каждый час.
func effectiveWakingHours(elapsed time.Duration) float64 {
	totalHours := elapsed.Hours()
	if totalHours <= 0 {
		return 0
	}
	if totalHours <= 24 {
		return totalHours
	}
	return 24 + (totalHours-24)*(16.0/24.0)
}

// calcAvgTimeBetween: среднее за месяц (day) или год (остальные), включая текущий интервал.
func calcAvgTimeBetween(habit models.Habit, relapses []models.Relapse, until time.Time) time.Duration {
	if until.Before(habit.OriginAt) {
		return 0
	}
	start := avgTimeWindowStart(habit, until)
	inWindow := filterSince(relapses, start, until)

	elapsed := until.Sub(start)
	var total time.Duration
	if habit.AvgRelapsesPeriod == models.PeriodDay {
		total = time.Duration(effectiveWakingHours(elapsed) * float64(time.Hour))
	} else {
		total = elapsed
	}
	intervals := len(inWindow) + 1
	return total / time.Duration(intervals)
}

// calcAvgPerPeriod: среднее срывов за период привычки с origin (lifetime count).
func calcAvgPerPeriod(habit models.Habit, relapseCount int, now time.Time) float64 {
	if now.Before(habit.OriginAt) {
		return 0
	}
	elapsed := now.Sub(habit.OriginAt)
	var totalDays float64
	if habit.AvgRelapsesPeriod == models.PeriodDay {
		totalDays = effectiveWakingHours(elapsed) / 16
	} else {
		totalDays = elapsed.Hours() / 24
	}
	if totalDays <= 0 {
		return 0
	}
	periodDays := habit.AvgRelapsesPeriod.Days()
	periods := totalDays / periodDays
	if periods <= 0 {
		return 0
	}
	avg := float64(relapseCount) / periods
	return math.Round(avg*100) / 100
}

func filterUntil(relapses []models.Relapse, until time.Time) []models.Relapse {
	var result []models.Relapse
	for _, r := range relapses {
		if r.RelapsedAt.Before(until) {
			result = append(result, r)
		}
	}
	return result
}

// filterSince keeps relapses in [start, until].
func filterSince(relapses []models.Relapse, start, until time.Time) []models.Relapse {
	var result []models.Relapse
	for _, r := range relapses {
		if !r.RelapsedAt.Before(start) && !r.RelapsedAt.After(until) {
			result = append(result, r)
		}
	}
	return result
}
