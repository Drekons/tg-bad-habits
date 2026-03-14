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
		timeTrend := trendIcon(st.AvgTimeTrend.Delta > 0, math.Abs(st.AvgTimeTrend.Delta), "ч")
		sb.WriteString(fmt.Sprintf("*%s*\n", escapeMarkdown(h.Name)))
		sb.WriteString(fmt.Sprintf("🕒 Прошло времени: %s\n", formatDuration(timeSince)))
		sb.WriteString(fmt.Sprintf("🕐 Последний срыв: %s\n", h.LastRelapseAt.Format(dateTimeLayout)))
		sb.WriteString(fmt.Sprintf("💰 Баланс: %s₽ %s\n", formatMoney(st.Balance), balanceTrend))
		sb.WriteString(fmt.Sprintf("📈 Среднее за %s: %.2f\n", h.AvgRelapsesPeriod.Label(), st.AvgPerPeriod))
		sb.WriteString(fmt.Sprintf("⏱ Среднее время между срывами: %s %s\n", formatDuration(st.AvgTimeBetween), timeTrend))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderStatsScreen builds the detailed stats screen for a single habit.
func RenderStatsScreen(h models.Habit, st service.HabitStats, last20 []models.Relapse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 *Статистика: %s*\n\n", escapeMarkdown(h.Name)))
	sb.WriteString(fmt.Sprintf("🕐 Последний срыв: %s\n", h.LastRelapseAt.Format(dateTimeLayout)))
	sb.WriteString(fmt.Sprintf("📅 Точка отсчёта: %s\n\n", h.OriginAt.Format(dateTimeLayout)))

	// Balance trend in currency
	balanceTrend := trendIcon(st.BalanceTrend.Delta > 0, math.Abs(st.BalanceTrend.Delta), "₽")
	sb.WriteString(fmt.Sprintf("💰 Баланс: *%s₽* %s\n", formatMoney(st.Balance), balanceTrend))

	// Avg per period
	sb.WriteString(fmt.Sprintf("📈 Среднее за %s: *%.2f*\n", h.AvgRelapsesPeriod.Label(), st.AvgPerPeriod))

	// Avg time between relapses - positive trend if time INCREASED (less frequent relapses)
	timeTrend := trendIcon(st.AvgTimeTrend.Delta > 0, math.Abs(st.AvgTimeTrend.Delta), "ч")
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

// trendIcon returns an emoji + delta string for a trend (green = positive, red = negative).
func trendIcon(better bool, delta float64, unit string) string {
	if delta == 0 {
		return "➡️ 0"
	}

	icon := "🔴" // negative trend
	if better {
		icon = "🟢" // positive trend
	}

	if delta < 0.01 {
		return fmt.Sprintf("%s <0.01%s", icon, unit)
	}

	return fmt.Sprintf("%s %.2f%s", icon, delta, unit)
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
