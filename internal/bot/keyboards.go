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

// mainInlineKeyboard builds the main screen inline keyboard: per habit [💥 Срыв | 📋 Меню], bottom [Меню].
func mainInlineKeyboard(habits []models.Habit) *tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, h := range habits {
		relapseData := "relapse:" + strconv.FormatInt(h.ID, 10)
		menuData := "habit_menu:" + strconv.FormatInt(h.ID, 10)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("💥 Срыв", relapseData),
			tgbotapi.NewInlineKeyboardButtonData(h.Name, menuData),
		})
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Меню", "main_menu"),
	})
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return &kb
}

// mainMenuReplyKeyboard — Reply для экрана «Выберите действие» (по callback main_menu).
func mainMenuReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить привычку"),
			tgbotapi.NewKeyboardButton("🏠 Перейти на главную"),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}

// habitMenuReplyKeyboard — Reply для экрана меню привычки: Срыв, Статистика, Назад.
func habitMenuReplyKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💥 Срыв"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("◀️ Назад"),
		),
	)
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
