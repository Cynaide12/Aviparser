package bot

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func InitBot(token string) *tele.Bot {
	bot, err := tele.NewBot(tele.Settings{Token: token, ParseMode: tele.ModeMarkdown})
	if err != nil {
		panic(fmt.Sprintf("ошибка при инициализации бота: %s", err))
	}

	return bot
}