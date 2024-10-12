package parser

import (
	"aviparser/internal/selectors"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func StartParse(ctx context.Context) {
	var productNodes []*cdp.Node
	url := "https://www.avito.ru/amurskaya_oblast_svobodnyy/kvartiry/sdam/posutochno/"
	err := chromedp.Run(ctx,
		chromedp.Navigate(fmt.Sprintf("%s%s", url, "?s=104")),
		chromedp.Nodes(".iva-item-content-rejJg", &productNodes, chromedp.ByQueryAll),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	var link string
	var links []string
	for _, node := range productNodes {
		var ok bool
		err = chromedp.Run(ctx,
			chromedp.AttributeValue(".iva-item-title-py3i_ a", "href", &link, &ok, chromedp.ByQuery, chromedp.FromNode(node)),
		)

		if err != nil {
			log.Fatal("Error:", err)
		}
		link = fmt.Sprintf("%s%s", "https://www.avito.ru", link)
		links = append(links, link)
	}
	fmt.Print(len(links))

	var apartments []Apartment
	wg := sync.WaitGroup{}
	for _, link := range links {
		wg.Add(1)
		apartment, err := ParseItemPage(ctx, link)
		if err != nil {
			log.Printf("Ошибка при парсинге группы: %s", err)
		}
		apartments = append(apartments, *apartment)
		wg.Done()
	}
	wg.Wait()
	saveApartmentsFromJson("apartments.json", apartments)
	fmt.Print("ЗАКОНЧИЛ")
}

func ParseItemPage(ctx context.Context, url string) (*Apartment, error) {
	var monthCalendars []*cdp.Node
	// Переход на страницу и открытие календаря
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(selectors.DatepickerButton),
		chromedp.Click(selectors.DatepickerButton),
		chromedp.WaitVisible(selectors.Datepicker),
		chromedp.Nodes(selectors.MonthCalendar, &monthCalendars, chromedp.ByQueryAll),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске календаря: %s", err.Error())
	}

	// Функция для поиска доступных дат внутри календаря месяца
	getAvailableDates := func(node *cdp.Node) ([]*cdp.Node, error) {
		var availableDates []*cdp.Node
		err = chromedp.Run(ctx,
			chromedp.Nodes(selectors.AvailableDate, &availableDates, chromedp.FromNode(node), chromedp.ByQueryAll),
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при поиске дат: %s", err.Error())
		}
		return availableDates, nil
	}


	//Получение свободныех дней с привязкой к месяцам
	avialabedDatesByMonth := make(map[string][]string)

	for _, monthNode := range monthCalendars {
		var monthName string
		err := chromedp.Run(ctx,
			chromedp.Text(selectors.MonthName, &monthName, chromedp.FromNode(monthNode), chromedp.ByQuery),
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при получении свободных дат месяца: %s", err.Error())
		}

		avialabedDates, err := getAvailableDates(monthNode)
		if err != nil {
			return nil, fmt.Errorf("ошибка при получении свободных дат месяца: %s", err.Error())
		}

		var avialabedDatesByString []string
		for _, dateNode := range avialabedDates {
			avialabedDatesByString = append(avialabedDatesByString, dateNode.Children[0].NodeValue)
		}

		avialabedDatesByMonth[monthName] = avialabedDatesByString
	}

	//Получение остальных полей
	var price, desc, title string
	var ok bool
	var apartmentTypes []*cdp.Node
	err = chromedp.Run(ctx,
		chromedp.AttributeValue(selectors.Price, "content", &price, &ok, chromedp.ByQuery),
		chromedp.Text(selectors.Description, &desc, chromedp.ByQuery),
		chromedp.Nodes(selectors.ApartmentType, &apartmentTypes, chromedp.ByQueryAll),
		chromedp.Text(selectors.Title, &title, chromedp.ByQuery),
	)
	apartmentType := apartmentTypes[len(apartmentTypes)-1].Children[0].NodeValue

	apartment := Apartment{
		Title:          title,
		Price:          price,
		Link:           url,
		Description:    desc,
		Type:           apartmentType,
		AvialableDates: avialabedDatesByMonth,
	}

	return &apartment, nil
}

func saveApartmentsFromJson(path string, apartments []Apartment) {
	jsonData, err := json.MarshalIndent(apartments, "", " ")
	if err != nil {
		log.Printf("ошибка при маршаллинге структуры апартаментов: %s", err.Error())
	}

	os.WriteFile(path, jsonData, 0644)
}
