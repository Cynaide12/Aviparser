package main

import (
	"aviparser/cmd/bot"
	"aviparser/internal/config"
	"aviparser/internal/handlers"
	"aviparser/internal/parser"
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg := config.MustLoad()
	AviBot := bot.AviBot{
		Bot:       bot.InitBot(cfg.BotToken),
		ChannelID: cfg.ChannelID,
	}
	c := cron.New()

	// ctx, acancel, cancel := chromedpContext()
	// go parser.StartParse(ctx, AviBot, cancel, acancel)

	if _, err := c.AddFunc("@every 1h", func() {
		ctx, acancel, cancel := chromedpContext()
		parser.StartParse(ctx, AviBot, cancel, acancel)
	}); err != nil {
		fmt.Println("Ошибка при добавлении задачи:", err)
		return
	}
	go c.Start()

	InitHandler(AviBot)

	AviBot.Bot.Start()

	defer AviBot.Bot.Stop()

}

func chromedpContext() (context.Context, context.CancelFunc, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"),
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("no-sandbox", true),
	)

	actx, acancel := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancel := chromedp.NewContext(actx)

	return ctx, acancel, cancel
}

func InitHandler(bot bot.AviBot){
	commandHandler := handlers.NemCommandHandler(&bot)

	bot.Bot.Handle("/getapartments", commandHandler.GetExcelFileHandler)
}