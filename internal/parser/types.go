package parser

type Apartment struct {
	Title  string `json:"name"`
	Price string `json:"price"`
	Link  string `json:"link"`
	Description  string `json:"description"`
	Type string `json:"type"`
	AvialableDates map[string][]string `json:"avialabledates"`
}

type ChangetApartments struct{
	NewPrices []Apartment
	NewAvialableDates []Apartment
	NewApartments []Apartment
}