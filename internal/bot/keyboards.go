package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/drek/tg-bad-habbits/internal/models"
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

// mainKeyboard builds the main screen keyboard dynamically from user habits.
func mainKeyboard(habits []models.Habit) tgbotapi.ReplyKeyboardMarkup {
	var rows [][]tgbotapi.KeyboardButton

	// One button per habit (max 2 per row for readability)
	for i := 0; i < len(habits); i += 2 {
		row := []tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton(habits[i].Name),
		}
		if i+1 < len(habits) {
			row = append(row, tgbotapi.NewKeyboardButton(habits[i+1].Name))
		}
		rows = append(rows, row)
	}

	// Action buttons
	rows = append(rows,
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 Статистика"),
			tgbotapi.NewKeyboardButton("➕ Добавить привычку"),
		),
	)

	kb := tgbotapi.NewReplyKeyboard(rows...)
	kb.ResizeKeyboard = true
	return kb
}

// confirmRelapseKeyboard shows Yes/No for relapse confirmation.
func confirmRelapseKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✅ Да"),
			tgbotapi.NewKeyboardButton("❌ Нет"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
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

// afterHabitCreatedKeyboard shows the "Go to main" button.
func afterHabitCreatedKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Перейти на главную"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// backKeyboard shows just a Back button.
func backKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// statsToMainKeyboard — выход из экранов статистики на главную (уникальный текст кнопки).
func statsToMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 На основной экран"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// statsHabitKeyboard builds the per-habit stats selection keyboard.
func statsHabitKeyboard(habits []models.Habit) tgbotapi.ReplyKeyboardMarkup {
	var rows [][]tgbotapi.KeyboardButton
	for _, h := range habits {
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📊 "+h.Name),
		))
	}
	rows = append(rows, tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("🏠 На основной экран"),
	))
	kb := tgbotapi.NewReplyKeyboard(rows...)
	kb.ResizeKeyboard = true
	return kb
}
