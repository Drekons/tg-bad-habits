package bot

import (
	"strconv"

	"github.com/drek/tg-bad-habbits/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// removeKeyboard returns a markup that forces the client to hide the custom keyboard.
func removeKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
		Selective:      false,
	}
}

// startKeyboard shows just the "Begin" button.
func startKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("▶️ Нажмите чтобы начать"),
		),
	)
}

// createFirstHabitKeyboard is shown after registration if no habits exist.
func createFirstHabitKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Создать первую вредную привычку"),
		),
	)
}

// mainInlineKeyboard: per habit [💥 Срыв | название→меню], bottom [Меню].
func mainInlineKeyboard(habits []models.Habit) *tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, h := range habits {
		id := strconv.FormatInt(h.ID, 10)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("💥 Срыв", "relapse:"+id+":main"),
			tgbotapi.NewInlineKeyboardButtonData(h.Name, "habit_menu:"+id),
		})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Меню", "main_menu"),
	})
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return &kb
}

// mainMenuInlineKeyboard — меню без Reply (callback’и не зависят от FSM).
func mainMenuInlineKeyboard() *tgbotapi.InlineKeyboardMarkup {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить привычку", "add_habit"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 На главную", "go_main"),
		),
	)
	return &kb
}

// habitMenuInlineKeyboard — меню привычки с habitID в callback.
func habitMenuInlineKeyboard(habitID int64) *tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatInt(habitID, 10)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💥 Срыв", "relapse:"+id+":menu"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "habit_stats:"+id),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 На главную", "go_main"),
		),
	)
	return &kb
}

// confirmRelapseInlineKeyboard — Да/Нет с habitID (и куда вернуться: main|menu).
func confirmRelapseInlineKeyboard(habitID int64, back string) *tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatInt(habitID, 10)
	if back != "menu" {
		back = "main"
	}
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да", "relapse_yes:"+id),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "relapse_no:"+id+":"+back),
		),
	)
	return &kb
}

// statsBackInlineKeyboard — назад в меню привычки.
func statsBackInlineKeyboard(habitID int64) *tgbotapi.InlineKeyboardMarkup {
	id := strconv.FormatInt(habitID, 10)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "habit_menu:"+id),
			tgbotapi.NewInlineKeyboardButtonData("🏠 На главную", "go_main"),
		),
	)
	return &kb
}

// periodKeyboard shows period selection buttons.
func periodKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("День"),
			tgbotapi.NewKeyboardButton("Месяц"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("3 месяца"),
			tgbotapi.NewKeyboardButton("Полгода"),
			tgbotapi.NewKeyboardButton("Год"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// defaultHabitNamesKeyboard shows preset habit name suggestions.
func defaultHabitNamesKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🚬 Курение"),
			tgbotapi.NewKeyboardButton("🍺 Алкоголь"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("☕ Кофе"),
			tgbotapi.NewKeyboardButton("🍬 Сладкое"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📱 Соцсети"),
			tgbotapi.NewKeyboardButton("🎮 Игры"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

