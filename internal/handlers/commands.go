package handlers

import (
	"aviparser/cmd/bot"
	"aviparser/internal/parser"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/360EntSecGroup-Skylar/excelize"
	tele "gopkg.in/telebot.v3"
)

var typePriority = map[string]int{
    "студии":       1,
    "1-комнатные":  2,
    "2-комнатные":  3,
    "3-комнатные":  4,
}

func NemCommandHandler(b *bot.AviBot) Handler {
	return Handler{
		Bot: b,
	}
}
func (h *Handler) GetExcelFileHandler(ctx tele.Context) error {
	createExcelFile()

	file := &tele.Document{File: tele.FromDisk("apartments.xlsx"), FileName: "apartments.xlsx"}
	return ctx.SendAlbum(tele.Album{file})
}


// Функция для чтения JSON файла
func readJSON(filename string) ([]parser.Apartment, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var apartments []parser.Apartment
	err = json.Unmarshal(file, &apartments)
	if err != nil {
		return nil, err
	}
	return apartments, nil
}

// Основная функция
func createExcelFile() {
    apartments, err := readJSON("apartments.json")
    if err != nil {
        log.Fatalf("Ошибка чтения JSON: %v", err)
    }
    
    sortApartments(apartments)

    fmt.Print(parser.FormatAvailableDates(apartments[0].AvialableDates))

    f := excelize.NewFile()

    // Заголовки столбцов
    f.SetCellValue("Sheet1", "A1", "Название")
    months := []string{"Октябрь", "Ноябрь"}

    colOffset := 1 // Столбец для календаря начинается с индекса 1 (т.е. B)
    
    // Устанавливаем заголовки для месяцев
    for _, month := range months {
        monthStartCol := getExcelColumn(colOffset)
        monthEndCol := getExcelColumn(colOffset + 30)
        f.MergeCell("Sheet1", fmt.Sprintf("%s1", monthStartCol), fmt.Sprintf("%s1", monthEndCol))
        f.SetCellValue("Sheet1", fmt.Sprintf("%s1", monthStartCol), month)

        // Добавляем дни в заголовки
        for day := 1; day <= 31; day++ {
            col := getExcelColumn(colOffset + day - 1)
            cell := fmt.Sprintf("%s2", col)
            f.SetCellValue("Sheet1", cell, fmt.Sprintf("%d", day))
        }

        colOffset += 31
    }

    // Добавляем столбец "Ссылка" после календаря
    f.SetCellValue("Sheet1", fmt.Sprintf("%s1", getExcelColumn(colOffset)), "Ссылка")

    // Добавляем данные о квартирах
    for i, apartment := range apartments {
        row := i + 3

        // Заполняем данные о квартире
        f.SetCellValue("Sheet1", fmt.Sprintf("A%d", row), apartment.Title)

        colOffset := 1 // Стартовый столбец для календаря (B)
        for _, month := range months {
            for day := 1; day <= 31; day++ {
                dayStr := fmt.Sprintf("%d", day)
                col := getExcelColumn(colOffset + day - 1)
                cell := fmt.Sprintf("%s%d", col, row)

                // Заполняем цену для свободных дат, если доступно
                if contains(apartment.AvialableDates[month], dayStr) {
                    f.SetCellValue("Sheet1", cell, apartment.Price)
                } else {
                    f.SetCellValue("Sheet1", cell, "")
                    style, _ := f.NewStyle(`{"fill":{"type":"pattern","color":["#FF0000"],"pattern":1}}`)
                    f.SetCellStyle("Sheet1", cell, cell, style)
                }
            }

            colOffset += 31
        }

        // Добавляем ссылку на квартиру после календаря
        f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", getExcelColumn(colOffset), row), apartment.Link)
    }
    
    // Устанавливаем ширину столбцов
    f.SetColWidth("Sheet1", "A", "A", 30)   // Ширина для названия
    f.SetColWidth("Sheet1", "B", "BK", 5)   // Ширина для календаря (можно отрегулировать, если нужно)
    f.SetColWidth("Sheet1", getExcelColumn(colOffset), getExcelColumn(colOffset), 5) // Ширина для ссылки

    // Автоматическое вычисление ширины для столбцов календаря
    for col := 1; col <= (colOffset-1); col++ {
        f.AutoFilter("Sheet1", getExcelColumn(col), fmt.Sprintf("%s%d", getExcelColumn(col), len(apartments)+2), "")
    }

    // Сохранение файла
    if err := f.SaveAs("apartments.xlsx"); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

}

// Функция для проверки, содержится ли день в списке доступных дат
func contains(s []string, str string) bool {
    for _, v := range s {
        if v == str {
            return true
        }
    }
    return false
}

func getExcelColumn(index int) string {
    if index < 26 {
        return string('A' + index)
    }
    first := 'A' + (index / 26) - 1
    second := 'A' + (index % 26)
    return string([]rune{rune(first), rune(second)})
}

// Функция для получения приоритета типа квартиры
func getTypePriority(apartmentType string) int {
    if priority, exists := typePriority[apartmentType]; exists {
        return priority
    }
    return 100 // Для типов, которых нет в карте, задаем высокий приоритет (в конец списка)
}

// Функция для сортировки массива квартир
func sortApartments(apartments []parser.Apartment) {
    sort.SliceStable(apartments, func(i, j int) bool {
        return getTypePriority(apartments[i].Type) < getTypePriority(apartments[j].Type)
    })
}
