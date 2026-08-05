package bot

import (
	"fmt"
	"html"
	"math"
	"strings"
	"time"

	"github.com/drek/tg-bad-habbits/internal/models"
	"github.com/drek/tg-bad-habbits/internal/service"
)

const dateTimeLayout = "02.01.2006 15:04"

// RenderMainScreen builds the text message for the main screen (HTML).
func RenderMainScreen(habits []models.Habit, stats []service.HabitStats) string {
	if len(habits) == 0 {
		return "У вас пока нет вредных привычек."
	}

	var sb strings.Builder
	sb.WriteString("📊 <b>Ваши привычки</b> (обновлено: " + time.Now().Format("15:04") + ")\n\n")

	for i, h := range habits {
		st := stats[i]
		timeSince := time.Since(h.LastRelapseAt)
		balanceTrend := trendIcon(st.BalanceTrend.Delta > 0, math.Abs(st.BalanceTrend.Delta), "₽")
		timeTrend := trendIconDuration(st.AvgTimeTrend.Delta > 0, math.Abs(st.AvgTimeTrend.Delta))
		nameLine := html.EscapeString(h.Name)
		if st.RelapsesInPeriod > 0 {
			nameLine = fmt.Sprintf("%s (x%d)", nameLine, st.RelapsesInPeriod)
		}
		sb.WriteString(fmt.Sprintf("<b>%s</b> - %s\n", nameLine, formatDuration(timeSince)))
		sb.WriteString(fmt.Sprintf("🕐 Последний: %s\n", h.LastRelapseAt.Format(dateTimeLayout)))
		sb.WriteString(fmt.Sprintf("💰 Баланс (год): %s₽ %s\n", formatMoney(st.Balance), balanceTrend))
		if st.AvgPerPeriod != 0 {
			avgLine := fmt.Sprintf("📈 Среднее за %s: %.2f", h.AvgRelapsesPeriod.Label(), st.AvgPerPeriod)
			if t := strings.TrimSpace(trendIcon(st.AvgPerPeriodTrend.Delta < 0, math.Abs(st.AvgPerPeriodTrend.Delta), "")); t != "" {
				avgLine += " " + t
			}
			sb.WriteString(avgLine + "\n")
		}
		sb.WriteString(fmt.Sprintf("⏱ Среднее время (%s): %s %s\n", avgTimeWindowLabel(h), formatDuration(st.AvgTimeBetween), timeTrend))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderStatsScreen builds the detailed stats screen for a single habit (HTML).
func RenderStatsScreen(h models.Habit, st service.HabitStats, last20 []models.Relapse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 <b>Статистика: %s</b>\n\n", html.EscapeString(h.Name)))
	sb.WriteString(fmt.Sprintf("🕐 Последний срыв: %s\n", h.LastRelapseAt.Format(dateTimeLayout)))
	sb.WriteString(fmt.Sprintf("📅 Точка отсчёта: %s\n", h.OriginAt.Format(dateTimeLayout)))
	sb.WriteString(fmt.Sprintf("📌 Кол-во срывов за %s: %d\n\n", h.AvgRelapsesPeriod.Label(), st.RelapsesInPeriod))

	balanceTrend := trendIcon(st.BalanceTrend.Delta > 0, math.Abs(st.BalanceTrend.Delta), "₽")
	sb.WriteString(fmt.Sprintf("💰 Баланс (год): <b>%s₽</b> %s\n", formatMoney(st.Balance), balanceTrend))

	avgLine := fmt.Sprintf("📈 Среднее за %s: <b>%.2f</b>", h.AvgRelapsesPeriod.Label(), st.AvgPerPeriod)
	if t := strings.TrimSpace(trendIcon(st.AvgPerPeriodTrend.Delta < 0, math.Abs(st.AvgPerPeriodTrend.Delta), "")); t != "" {
		avgLine += " " + t
	}
	sb.WriteString(avgLine + "\n")

	timeTrend := trendIconDuration(st.AvgTimeTrend.Delta > 0, math.Abs(st.AvgTimeTrend.Delta))
	sb.WriteString(fmt.Sprintf("⏱ Среднее между срывами (%s): <b>%s</b> %s\n\n", avgTimeWindowLabel(h), formatDuration(st.AvgTimeBetween), timeTrend))

	sb.WriteString("📋 <b>Последние срывы:</b>\n")
	if len(last20) == 0 {
		sb.WriteString("  — нет записей\n")
	} else {
		for i, r := range last20 {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, r.RelapsedAt.Format(dateTimeLayout)))
		}
	}

	return sb.String()
}

func trendIcon(better bool, delta float64, unit string) string {
	if delta == 0 {
		return ""
	}

	icon := "↓"
	if better {
		icon = "↑"
	}

	if delta < 0.01 {
		return fmt.Sprintf("%s &lt;0.01%s", icon, unit)
	}

	return fmt.Sprintf("%s %.2f%s", icon, delta, unit)
}

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

func formatMoney(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func avgTimeWindowLabel(h models.Habit) string {
	if h.AvgRelapsesPeriod == models.PeriodDay {
		return "мес"
	}
	return "год"
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	if d < time.Minute {
		if d == 0 {
			return "н/д"
		}
		return "1м"
	}
	totalMin := int(d / time.Minute)
	days := totalMin / (24 * 60)
	hours := (totalMin % (24 * 60)) / 60
	minutes := totalMin % 60

	if days > 0 {
		return fmt.Sprintf("%dд %dч %dм", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	}
	return fmt.Sprintf("%dм", minutes)
}
