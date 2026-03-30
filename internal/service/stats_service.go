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

// Calc returns full HabitStats for a habit given all its relapses and the reference time.
func (s *StatsService) Calc(habit models.Habit, relapses []models.Relapse, now time.Time) HabitStats {
	// "Yesterday" = start of today (00:00:00)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	relapsesUntilYesterday := filterUntil(relapses, startOfToday)

	balanceNow := calcBalance(habit, relapses, now)
	balanceYesterday := calcBalance(habit, relapsesUntilYesterday, startOfToday)

	avgTimeNow := calcAvgTimeBetween(habit, relapses, now)
	avgTimeYesterday := calcAvgTimeBetween(habit, relapsesUntilYesterday, startOfToday)

	balanceDelta := balanceNow - balanceYesterday
	avgTimeDelta := avgTimeNow - avgTimeYesterday
	// При нуле срывов тренд времени = просто "часы с полуночи", что неинформативно — показываем 0
	if len(relapses) == 0 {
		avgTimeDelta = 0
	}

	avgPerNow := calcAvgPerPeriod(habit, relapses, now)
	avgPerYesterday := calcAvgPerPeriod(habit, relapsesUntilYesterday, startOfToday)
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
			Up:       avgPerDelta > 0, // рост среднего числа срывов — хуже
		},
		RelapsesInPeriod: countRelapsesInPeriod(habit, relapses, now),
	}
}

// countRelapsesInPeriod returns the number of relapses in the current period for the habit.
// Day = from 00:00 today; month = current calendar month; 3m/6m/year = last 90/180/365 days.
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

// calcBalance computes: potentialLoss(until) - realLoss(relapses).
// For habits with period "day", uses effective (waking) days: see effectiveWakingHours.
func calcBalance(habit models.Habit, relapses []models.Relapse, until time.Time) float64 {
	if until.Before(habit.OriginAt) {
		return 0
	}
	elapsed := until.Sub(habit.OriginAt)
	// Show intermediate balance even right after creation: use at least 1 minute so balance is not stuck at 0.
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
	realLoss := float64(len(relapses)) * habit.CostPerRelapse
	balance := math.Round((potentialLoss-realLoss)*100) / 100
	// Когда срывов не было и время прошло, баланс не должен быть 0 (промежуточный баланс).
	if len(relapses) == 0 && until.After(habit.OriginAt) && balance <= 0 {
		daysSince = math.Max(daysSince, 1)
		avgPerDayMin := habit.AvgRelapsesCount / habit.AvgRelapsesPeriod.Days()
		if avgPerDayMin <= 0 {
			avgPerDayMin = 1.0 / 365 // минимум 1 срыв в год для отображения
		}
		if habit.CostPerRelapse > 0 {
			potentialLoss = avgPerDayMin * daysSince * habit.CostPerRelapse
			return math.Round(potentialLoss*100) / 100
		}
	}
	return balance
}

// effectiveWakingHours returns waking hours for a daily habit: first 24h from origin count
// fully; after that, each real hour contributes 16/24 waking hours (8h sleep spread smoothly
// over each day after the registration day).
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

// calcAvgTimeBetween computes mean duration of all intervals including the ongoing one up to 'until'.
// For period "day", total time is effective waking time (8h sleep per full day after registration day excluded).
func calcAvgTimeBetween(habit models.Habit, relapses []models.Relapse, until time.Time) time.Duration {
	if until.Before(habit.OriginAt) {
		return 0
	}
	elapsed := until.Sub(habit.OriginAt)
	var total time.Duration
	if habit.AvgRelapsesPeriod == models.PeriodDay {
		total = time.Duration(effectiveWakingHours(elapsed) * float64(time.Hour))
	} else {
		total = elapsed
	}
	intervals := len(relapses) + 1
	return total / time.Duration(intervals)
}

// calcAvgPerPeriod computes real average relapses per the habit's period since origin.
// For period "day", elapsed "days" match balance (effective waking / 16).
func calcAvgPerPeriod(habit models.Habit, relapses []models.Relapse, now time.Time) float64 {
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
	avg := float64(len(relapses)) / periods
	return math.Round(avg*100) / 100
}

// filterUntil returns only relapses that occurred strictly before the cutoff.
func filterUntil(relapses []models.Relapse, until time.Time) []models.Relapse {
	var result []models.Relapse
	for _, r := range relapses {
		if r.RelapsedAt.Before(until) {
			result = append(result, r)
		}
	}
	return result
}
