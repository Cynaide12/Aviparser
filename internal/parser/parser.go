package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"log"
	"os"
)

func StartParse() {
	// to keep track of all the scraped objects
	var products []Apartment

	// initialize a controllable Chrome instance
	ctx, cancel := chromedp.NewContext(
		context.Background(),
	)
	// to release the browser resources when
	// it is no longer needed
	defer cancel()

	// browser automation logic
	var productNodes []*cdp.Node
	url := "https://www.avito.ru/amurskaya_oblast_svobodnyy/kvartiry/sdam/posutochno/"
	err := chromedp.Run(ctx,
		chromedp.Navigate(fmt.Sprintf("%s%s", url, "?s=104")),
		chromedp.Nodes(".iva-item-content-rejJg", &productNodes, chromedp.ByQueryAll),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// scraping logic
	var name, price, link string
	var links []string
	for _, node := range productNodes {
		// extract data from the product HTML node
		var ok bool
		err = chromedp.Run(ctx,
			chromedp.Text(".iva-item-title-py3i_ a", &name, chromedp.ByQuery, chromedp.FromNode(node)),
			chromedp.AttributeValue(".iva-item-title-py3i_ a", "href", &link, &ok, chromedp.ByQuery, chromedp.FromNode(node)),
			chromedp.Text(".iva-item-descriptionStep-C0ty1", &price, chromedp.ByQuery, chromedp.FromNode(node)),
		)

		if err != nil {
			log.Fatal("Error:", err)
		}
		link = fmt.Sprintf("%s%s", "https://www.avito.ru", link)
		product := Apartment{
			Title: name,
			Price: price,
			Link:  link,
		}
		links = append(links, link)
		products = append(products, product)
	}

	for i, link := range links {
		var desc string
		fmt.Print(link)
		err = chromedp.Run(ctx,
			chromedp.Navigate(link),
			chromedp.WaitVisible(".style-item-description-pL_gy", chromedp.ByQuery),
			chromedp.Text(".style-item-description-pL_gy p", &desc, chromedp.ByQuery),
		)
		if err != nil {
			panic(err)
		}
		products[i].Description = desc
	}

	// Выводим результат
	fmt.Println(products)

	// Сохраняем данные в JSON файл
	jsonData, err := json.MarshalIndent(products, "", " ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("avito.json", jsonData, 0644); err != nil {
		panic(fmt.Sprintf("не получилось записать %s", err))
	}
}

func ParseItemPage(ctx context.Context, url string) {
	var monthCalendars []*cdp.Node

	// Переход на страницу и открытие календаря
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(".datepicker-range-input-_C1z4"),
		chromedp.Click(".datepicker-range-input-_C1z4"),
		chromedp.WaitVisible("*[data-marker='datepicker']"),
		chromedp.Nodes(".styles-module-calendar-I_fDT", &monthCalendars, chromedp.ByQueryAll),
	)
	if err != nil {
		log.Fatalf("Ошибка при поиске календаря: %s", err.Error())
	}

	// Функция для поиска доступных дат внутри календаря месяца
	getAvailableDates := func(node *cdp.Node) ([]*cdp.Node, error) {
		var availableDates []*cdp.Node
		err = chromedp.Run(ctx,
			chromedp.Nodes(".styles-module-day_hoverable-_qBAt", &availableDates, chromedp.FromNode(node), chromedp.ByQueryAll),
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при поиске дат: %s", err.Error())
		}
		return availableDates, nil
	}

	avialabedDatesByMonth := make(map[string][]string)

	for _, monthNode := range monthCalendars {
		var monthName string
		err := chromedp.Run(ctx,
			chromedp.Text(".styles-module-title_month-Ik8nn", &monthName, chromedp.FromNode(monthNode), chromedp.ByQuery),
		)
		if err != nil {
			log.Fatalf("ошибка при получении свободных дат месяца: %s", err.Error())
		}

		avialabedDates, err := getAvailableDates(monthNode)
		if err != nil {
			log.Fatalf("ошибка при получении свободных дат месяца: %s", err.Error())
		}

		fmt.Printf("\nМЕСЯЦ - %s", monthName)

		var avialabedDatesByString []string
		for _, dateNode := range avialabedDates {
			avialabedDatesByString = append(avialabedDatesByString, dateNode.Children[0].NodeValue)
		}

		avialabedDatesByMonth[monthName] = avialabedDatesByString
	}

	var price, desc, title string
	var ok bool
	var apartmentTypes []*cdp.Node
	//получение цены и описания
	err = chromedp.Run(ctx,
		chromedp.AttributeValue("*[data-marker='item-view/item-price']", "content", &price, &ok, chromedp.ByQuery),
		chromedp.Text("*[data-marker='item-view/item-description']", &desc, chromedp.ByQuery),
		chromedp.Nodes(".breadcrumbs-linkWrapper-jZP0j span", &apartmentTypes, chromedp.ByQueryAll),
		chromedp.Text(".style-titleWrapper-Hmr_5 h1", &title, chromedp.ByQuery),
	)
	apartmentType := apartmentTypes[len(apartmentTypes)-1].Children[0].NodeValue

	apartment := Apartment{
		Title:       title,
		Price:       price,
		Link:        url,
		Description: desc,
		Type:        apartmentType,
		AvialableDates: avialabedDatesByMonth,
	}

	saveApartmentFromJson("test.json", apartment)


	fmt.Print(len(monthCalendars))
}

func saveApartmentFromJson(path string, apartment Apartment) {
	jsonData, err := json.MarshalIndent(apartment, "", " ")
	if err != nil {
		log.Printf("ошибка при маршаллинге структуры апартаментов: %s", err.Error())
	}

	os.WriteFile(path, jsonData, 0644)
}
