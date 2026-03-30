package bot

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/drek/tg-bad-habbits/internal/models"
	"github.com/drek/tg-bad-habbits/internal/service"
)

const dateTimeLayout = "02.01.2006 15:04"

// RenderMainScreen builds the text message for the main screen.
func RenderMainScreen(habits []models.Habit, stats []service.HabitStats) string {
	if len(habits) == 0 {
		return "У вас пока нет вредных привычек."
	}

	var sb strings.Builder
	sb.WriteString("📊 *Ваши привычки* (обновлено: " + time.Now().Format("15:04") + ")\n\n")

	for i, h := range habits {
		st := stats[i]
		timeSince := time.Since(h.LastRelapseAt)
		balanceTrend := trendIcon(st.BalanceTrend.Delta > 0, math.Abs(st.BalanceTrend.Delta), "₽")
		timeTrend := trendIconDuration(st.AvgTimeTrend.Delta > 0, math.Abs(st.AvgTimeTrend.Delta))
		nameLine := escapeMarkdown(h.Name)
		if st.RelapsesInPeriod > 0 {
			nameLine = fmt.Sprintf("%s (x%d)", nameLine, st.RelapsesInPeriod)
		}
		sb.WriteString(fmt.Sprintf("*%s* - %s\n", nameLine, formatDuration(timeSince)))
		sb.WriteString(fmt.Sprintf("🕐 Последний: %s\n", h.LastRelapseAt.Format(dateTimeLayout)))
		sb.WriteString(fmt.Sprintf("💰 Баланс: %s₽ %s\n", formatMoney(st.Balance), balanceTrend))
		if st.AvgPerPeriod != 0 {
			avgLine := fmt.Sprintf("📈 Среднее за %s: %.2f", h.AvgRelapsesPeriod.Label(), st.AvgPerPeriod)
			if t := strings.TrimSpace(trendIcon(st.AvgPerPeriodTrend.Delta < 0, math.Abs(st.AvgPerPeriodTrend.Delta), "")); t != "" {
				avgLine += " " + t
			}
			sb.WriteString(avgLine + "\n")
		}
		sb.WriteString(fmt.Sprintf("⏱ Среднее время: %s %s\n", formatDuration(st.AvgTimeBetween), timeTrend))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderStatsScreen builds the detailed stats screen for a single habit.
func RenderStatsScreen(h models.Habit, st service.HabitStats, last20 []models.Relapse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 *Статистика: %s*\n\n", escapeMarkdown(h.Name)))
	sb.WriteString(fmt.Sprintf("🕐 Последний срыв: %s\n", h.LastRelapseAt.Format(dateTimeLayout)))
	sb.WriteString(fmt.Sprintf("📅 Точка отсчёта: %s\n", h.OriginAt.Format(dateTimeLayout)))
	sb.WriteString(fmt.Sprintf("📌 Кол-во срывов за %s: %d\n\n", h.AvgRelapsesPeriod.Label(), st.RelapsesInPeriod))

	// Balance trend in currency
	balanceTrend := trendIcon(st.BalanceTrend.Delta > 0, math.Abs(st.BalanceTrend.Delta), "₽")
	sb.WriteString(fmt.Sprintf("💰 Баланс: *%s₽* %s\n", formatMoney(st.Balance), balanceTrend))

	// Меньше срывов за период лучше (↑ при отрицательной дельте).
	avgLine := fmt.Sprintf("📈 Среднее за %s: *%.2f*", h.AvgRelapsesPeriod.Label(), st.AvgPerPeriod)
	if t := strings.TrimSpace(trendIcon(st.AvgPerPeriodTrend.Delta < 0, math.Abs(st.AvgPerPeriodTrend.Delta), "")); t != "" {
		avgLine += " " + t
	}
	sb.WriteString(avgLine + "\n")

	// Avg time between relapses - positive trend if time INCREASED (less frequent relapses)
	timeTrend := trendIconDuration(st.AvgTimeTrend.Delta > 0, math.Abs(st.AvgTimeTrend.Delta))
	sb.WriteString(fmt.Sprintf("⏱ Среднее между срывами: *%s* %s\n\n", formatDuration(st.AvgTimeBetween), timeTrend))

	// Last 20 relapses
	sb.WriteString("📋 *Последние срывы:*\n")
	if len(last20) == 0 {
		sb.WriteString("  — нет записей\n")
	} else {
		for i, r := range last20 {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, r.RelapsedAt.Format(dateTimeLayout)))
		}
	}

	return sb.String()
}

// trendIcon returns up/down arrow + delta string for a trend.
func trendIcon(better bool, delta float64, unit string) string {
	if delta == 0 {
		return ""
	}

	icon := "↓" // отрицательный тренд
	if better {
		icon = "↑" // положительный тренд
	}

	if delta < 0.01 {
		return fmt.Sprintf("%s <0.01%s", icon, unit)
	}

	return fmt.Sprintf("%s %.2f%s", icon, delta, unit)
}

// trendIconDuration returns arrow + delta formatted as duration (e.g. "↑ 6м", "↓ 1ч 30м").
func trendIconDuration(better bool, deltaHours float64) string {
	if deltaHours == 0 {
		return ""
	}
	d := time.Duration(deltaHours * float64(time.Hour))
	if d < time.Minute && d > 0 {
		d = time.Minute
	}
	icon := "↓"
	if better {
		icon = "↑"
	}
	return icon + " " + formatDuration(d)
}

// formatMoney formats a float as a readable money string.
func formatMoney(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "н/д"
	}
	total := int(d.Hours())
	days := total / 24
	hours := total % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dд %dч %dм", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	}
	return fmt.Sprintf("%dм", minutes)
}

// escapeMarkdown escapes special Markdown characters for Telegram.
func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"`", "\\`",
	)
	return replacer.Replace(s)
}
