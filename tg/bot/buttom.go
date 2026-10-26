package bot

import (
	tgbotapi "github.com/kirugan/telegram-bot-api/v5"
)

func createOrderKeybord() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👍", "good"),
			tgbotapi.NewInlineKeyboardButtonData("👎", "bad"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отзыв", "order"),
		),
	)
}

func createGoodKeybord() tgbotapi.InlineKeyboardMarkup {
	good := tgbotapi.NewInlineKeyboardButtonData("👍", "good")
	good.Style = "success"

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			good,
			tgbotapi.NewInlineKeyboardButtonData("👎", "bad"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отзыв", "order"),
		),
	)
}

func createBadKeybord() tgbotapi.InlineKeyboardMarkup {
	bad := tgbotapi.NewInlineKeyboardButtonData("👎", "bad")
	bad.Style = "danger"

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👍", "good"),
			bad,
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отзыв", "order"),
		),
	)
}

func createRegistrKeybord() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅Принять", "accept"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Подписаться", conf.TG_CHANNEL_URL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Проверить подписку", "check subscription"),
		),
	)
}

func createCommandKeybord() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Транскрибация", "transcride"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Моя статистика", "stat"),
		),
	)
}

func createMenuKeybord() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Меню", "show_menu"),
		),
	)
}

func createErrorWorkKeybord() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Исправить ошибку", "fix_error"),
		),
	)
}
