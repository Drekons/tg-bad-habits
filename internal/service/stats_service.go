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
	Balance        float64
	BalanceTrend   TrendData
	AvgTimeBetween time.Duration
	AvgTimeTrend   TrendData
	AvgPerPeriod   float64 // real average relapses per habit's period
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
		AvgPerPeriod: calcAvgPerPeriod(habit, relapses, now),
	}
}

// calcBalance computes: potentialLoss(until) - realLoss(relapses).
// For habits with period "day", uses effective (waking) days: 8h sleep per full day
// after the registration day is excluded. Registration day is not reduced.
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

// effectiveWakingHours returns waking hours for a daily habit: total time minus 8h sleep
// per full 24h day after the first day (registration day is not reduced).
func effectiveWakingHours(elapsed time.Duration) float64 {
	totalHours := elapsed.Hours()
	if totalHours <= 0 {
		return 0
	}
	fullDays := int(totalHours / 24)
	sleepingHours := 0.0
	if fullDays >= 1 {
		sleepingHours = 8 * float64(fullDays-1)
	}
	return totalHours - sleepingHours
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
func calcAvgPerPeriod(habit models.Habit, relapses []models.Relapse, now time.Time) float64 {
	if now.Before(habit.OriginAt) {
		return 0
	}
	totalDays := now.Sub(habit.OriginAt).Hours() / 24
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
