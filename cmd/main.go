package main

import (
	"aviparser/internal/parser"
	"context"

	"github.com/chromedp/chromedp"
)

func main() {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),    // Отключаем headless режим
		chromedp.Flag("disable-gpu", true), // Включаем графический процессор
	)

	actx, acancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer acancel()

	ctx, cancel := chromedp.NewContext(actx)
	defer cancel()
	parser.ParseItemPage(ctx, "https://www.avito.ru/amurskaya_oblast_svobodnyy/kvartiry/kvartira-studiya_36_m_15_et._1823244811?guestsDetailed=%7B%22version%22%3A1%2C%22totalCount%22%3A1%2C%22adultsCount%22%3A1%2C%22children%22%3A%5B%5D%7D&calendar=true")
}
