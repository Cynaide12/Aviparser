package parser

import (
	"aviparser/cmd/bot"
	"aviparser/internal/selectors"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	tele "gopkg.in/telebot.v3"
)

func StartParse(ctx context.Context, AviBot bot.AviBot, cancel, acancel context.CancelFunc) {
	defer cancel()
	defer acancel()

	var AllCounts string
	var apartmentsLink []string
	//Заходим в каталог и парсим квартиры
	url := "https://www.avito.ru/amurskaya_oblast_svobodnyy/kvartiry/sdam/posutochno/"

	ctxWithTimeout, _ := context.WithTimeout(ctx, 10*time.Minute)

	err := chromedp.Run(ctxWithTimeout,
		chromedp.Navigate(fmt.Sprintf("%s%s", url, "?s=104")),
	)
	if err != nil {
		log.Printf("ошибка при заходе на страницу: %s", err)
	}

	chromedp.Run(ctxWithTimeout,
		chromedp.Text(selectors.AllCount, &AllCounts, chromedp.ByQueryAll),
	)

	fmt.Printf("\nPAGINATION %v", len(AllCounts))
	allCountsInt, _ := strconv.Atoi(AllCounts)
	stepCount := int(math.Ceil(float64(allCountsInt) / 50.0))
	for i := 1; i <= stepCount; i++ {

		err := chromedp.Run(ctxWithTimeout,
			chromedp.Navigate(fmt.Sprintf("%s?p=%d%s", url, i, "&s=104")),
		)
		if err != nil {
			log.Printf("ошибка при заходе на страницу: %s", err)
		}

		var productNodes []*cdp.Node
		var productContainer []*cdp.Node
		chromedp.Run(ctx,
			chromedp.Nodes(".items-items-kAJAg", &productContainer, chromedp.ByQuery),
		)

		if err != nil {
			log.Printf("ошибка при поиске контейнера квартир: %s", err)
		}

		if len(productContainer) < 1 {
			log.Printf("не найден контейнер квартир. вероятно блок IP")
			return
		}

		//Ищем квартиры
		err = chromedp.Run(ctx,
			chromedp.Nodes(".iva-item-content-rejJg", &productNodes, chromedp.FromNode(productContainer[0]), chromedp.ByQueryAll),
		)
		if err != nil {
			log.Printf("ошибка при поиске квартир: %s", err)
			return
		}

		//Собираем ссылки
		var link string
		for _, node := range productNodes {
			var ok bool
			err = chromedp.Run(ctx,
				chromedp.AttributeValue(".iva-item-title-py3i_ a", "href", &link, &ok, chromedp.ByQuery, chromedp.FromNode(node)),
			)

			if err != nil {
				log.Printf("ошибка при поиске ссылок: %s", err)
				return
			}
			link = fmt.Sprintf("%s%s", "https://www.avito.ru", link)
			apartmentsLink = append(apartmentsLink, link)
		}
	}

	fmt.Printf("\nLINKS %d", len(apartmentsLink))

	var apartments []Apartment

	for i, link := range apartmentsLink {

		apartment, err := ParseItemPage(ctx, link)
		if err != nil {
			log.Printf("Ошибка при парсинге страницы: %s", err)
		} else {
			apartments = append(apartments, *apartment)
		}
		fmt.Printf("\n ЗАКОНЧИЛ С %d КВАРТИРОЙ", i + 1)
		time.Sleep(1 * time.Second)

	}

	//сохраняем новые квартиры в json, чтобы избежжать проблем с кодировкой при сравнении
	if len(apartments) < 1{
		return
	}

	SaveApartmentsFromJson("newApartments.json", apartments)

	//загружаем
	apartments, err = LoadApartments("newApartments.json")
	if err != nil {
		log.Printf("ошибка при загрузке квартир %s", err)
	}

	prevApartmentsBySlice, err := LoadApartments("apartments.json")
	if err != nil {
		log.Printf("ошибка при загрузке квартир %s", err)
	}

	prevApartmentsByMap := GetApartmentsByMap(prevApartmentsBySlice)
	var changedApartments ChangetApartments
	for _, newApartment := range apartments {
		changedApartments, _ = compareApartments(newApartment, prevApartmentsByMap, changedApartments)
	}

	//после получения прошлых квартир можно сохранить новые
	SaveApartmentsFromJson("apartments.json", apartments)

	sendMessage(AviBot, changedApartments)
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

		ctxWithTimeout, _ := context.WithTimeout(ctx, 500*time.Millisecond)

		err = chromedp.Run(ctxWithTimeout,
			chromedp.Nodes(selectors.AvailableDate, &availableDates, chromedp.FromNode(node), chromedp.ByQueryAll),
		)
		return availableDates, nil
	}

	//Получение свободныех дней с привязкой к месяцам
	avialabedDatesByMonth := make(map[string][]string)

	for _, monthNode := range monthCalendars {
		var monthName string
		err := chromedp.Run(ctx,
			chromedp.WaitVisible(selectors.MonthName),
			chromedp.Text(selectors.MonthName, &monthName, chromedp.FromNode(monthNode), chromedp.ByQuery),
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при получении названия месяца: %s", err.Error())
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
	if len(apartmentTypes) < 2 {
		return nil, nil
	}
	apartmentType := apartmentTypes[len(apartmentTypes)-1].Children[0].NodeValue

	apartment := Apartment{
		Title:          normalizeString(title),
		Price:          normalizeString(price),
		Link:           url,
		Description:    normalizeString(desc),
		Type:           normalizeString(apartmentType),
		AvialableDates: avialabedDatesByMonth,
	}

	return &apartment, nil
}

func sendMessage(bot bot.AviBot, changedApartments ChangetApartments) {
	var messages []string
	for _, newPriceApartment := range changedApartments.NewPrices {
		template := getTemplateMessage(newPriceApartment)
		message := fmt.Sprintf("Новая цена у квартиры\n %s", template)
		messages = append(messages, message)
	}
	for _, newAvialableApartment := range changedApartments.NewAvialableDates {
		template := getTemplateMessage(newAvialableApartment)
		message := fmt.Sprintf("Изменились свободные даты у квартиры\n %s", template)
		messages = append(messages, message)
	}
	for _, newApartments := range changedApartments.NewApartments {
		template := getTemplateMessage(newApartments)
		message := fmt.Sprintf("Новая квартира\n %s", template)
		messages = append(messages, message)
	}

	for _, msg := range messages {
		_, err := bot.Bot.Send(tele.ChatID(bot.ChannelID), msg, &tele.SendOptions{ParseMode: tele.ModeHTML})
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
		time.Sleep(3 * time.Second)
	}
}

func FormatAvailableDates(dates map[string][]string) string {
    var result []string
    for month, days := range dates {
        dayString := fmt.Sprintf("%s: %v", month, days)
        result = append(result, dayString)
    }
    return strings.Join(result, "; ")
}

func getTemplateMessage(apartment Apartment) string {
	avialabledDates := FormatAvailableDates(apartment.AvialableDates)
	return fmt.Sprintf("Квартира: %s,\n Ссылка: %s,\n Тип квартиры: %s,\n Цена: %s,\n Описание: %s,\n Свободные даты: %v",
		apartment.Title,
		apartment.Link,
		apartment.Type,
		apartment.Price,
		apartment.Description,
		avialabledDates)
}

func SaveApartmentsFromJson(path string, apartments []Apartment) {
	jsonData, err := json.Marshal(apartments)
	if err != nil {
		log.Printf("ошибка при маршаллинге структуры апартаментов: %s", err.Error())
	}

	os.WriteFile(path, jsonData, 0644)
}

func LoadApartments(path string) ([]Apartment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка при открытии файла квартир: %s", err.Error())
	}

	var apartments []Apartment
	if err := json.Unmarshal(data, &apartments); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании json в структуру квартир: %s", err)
	}

	return apartments, nil
}

func GetApartmentsByMap(apartments []Apartment) map[string]Apartment {
	apartmentsMap := make(map[string]Apartment)
	for _, apartment := range apartments {
		apartmentsMap[apartment.Link] = apartment
	}
	fmt.Printf("\nМАПА - %d", len(apartmentsMap))
	return apartmentsMap
}

// Сравнение двух квартир
func compareApartments(newApartment Apartment, prevApartmentsByMap map[string]Apartment, changedApartments ChangetApartments) (ChangetApartments, bool) {
	prevApartment, ok := prevApartmentsByMap[newApartment.Link]
	isNew := false
	if !ok {
		changedApartments.NewApartments = append(changedApartments.NewApartments, newApartment)
		isNew = true
	}
	if (normalizeString(prevApartment.Price) != normalizeString(newApartment.Price)) && !isNew {
		changedApartments.NewPrices = append(changedApartments.NewPrices, newApartment)
	}

	if !areMapsEqual(prevApartment.AvialableDates, newApartment.AvialableDates) && !isNew {
		changedApartments.NewAvialableDates = append(changedApartments.NewAvialableDates, newApartment)
	}

	return changedApartments, isNew
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")
	return removeNonPrintable(s)
}

// Функция для удаления непечатаемых символов
func removeNonPrintable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}

// Сравнение двух мап
func areMapsEqual(map1, map2 map[string][]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, val1 := range map1 {
		val2, ok := map2[key]
		if !ok {
			return false
		}

		if len(val1) != len(val2) {
			return false
		}

		for i := range val1 {
			if val1[i] != val2[i] {
				return false
			}
		}
	}

	return true
}
