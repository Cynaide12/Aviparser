package main

import (
	"aviparser/internal/parser"
	"context"
	"log"

	"github.com/chromedp/chromedp"
)

func main() {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
	)

	actx, acancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer acancel()

	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()
	// parser.StartParse(ctx)
	_, err := parser.ParseItemPage(ctx, "https://www.avito.ru/amurskaya_oblast_svobodnyy/kvartiry/kvartira-studiya_21_m_33_et._4465926250?guestsDetailed=%7B%22version%22%3A1%2C%22totalCount%22%3A2%2C%22adultsCount%22%3A2%2C%22children%22%3A%5B%5D%7D")
	if err != nil{
		log.Fatal(err)
	}
}
