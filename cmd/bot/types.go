package bot

import tele "gopkg.in/telebot.v3"

type AviBot struct {
	Bot *tele.Bot
	ChannelID int64
}