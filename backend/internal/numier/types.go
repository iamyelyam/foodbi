package numier

// APIResponse is the standard wrapper for all NUMIER API responses.
type APIResponse[T any] struct {
	Result     T      `json:"result"`
	Response   bool   `json:"response"`
	TotalPages int    `json:"totalpages"`
	Message    string `json:"message"`
}

// Locale represents a business/establishment linked to the NUMIER company.
type Locale struct {
	ID                string `json:"id"`
	EstablishmentName string `json:"establishmentName"`
}

// Sale represents a single sales ticket from NUMIER.
type Sale struct {
	Serie             string      `json:"Serie"`
	Number            string      `json:"Number"`
	TaxDocumentNumber string      `json:"TaxDocumentNumber"`
	BusinessDay       string      `json:"BusinessDay"`
	VatIncluded       bool        `json:"VatIncluded"`
	Date              string      `json:"Date"`
	Pos               SalePos     `json:"Pos"`
	Workplace         SalePlace   `json:"Workplace"`
	Section           SaleSection `json:"Section"`
	User              SaleUser    `json:"User"`
	NumDiners         int         `json:"NumDiners"`
	Channel           string      `json:"Channel"`
	DocumentType      string      `json:"DocumentType"`
	InvoiceItems      []SaleItem  `json:"InvoiceItems"`
	Payments          string      `json:"Payments"`
	Totals            SaleTotals  `json:"Totals"`
}

type SalePos struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

type SalePlace struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

type SaleSection struct {
	SectionName   string `json:"sectionName"`
	SectionNumber string `json:"sectionNumber"`
}

type SaleUser struct {
	UserCode string `json:"UserCode"`
}

type SaleItem struct {
	IDProduct   string        `json:"idProduct"`
	Name        string        `json:"name"`
	IDCategory  string        `json:"idCategory"`
	Units       string        `json:"units"`
	Price       string        `json:"price"`
	Amount      string        `json:"amount"`
	VatType     string        `json:"vatType"`
	Subproducts []Subproduct  `json:"subproducts"`
}

type Subproduct struct {
	IDSubproduct string  `json:"idSubproduct"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`      // "O" = optional, "E" = extra, "B" = base
	Variation    float64 `json:"variation"`
}

type SaleTotals struct {
	GrossAmount    float64            `json:"GrossAmount"`
	NetAmount      float64            `json:"NetAmount"`
	VatAmount      float64            `json:"VatAmount"`
	SurchargeAmount float64           `json:"SurchargeAmount"`
	Taxes          map[string]TaxInfo `json:"Taxes"`
}

type TaxInfo struct {
	NetAmount float64 `json:"NetAmount"`
	VatAmount float64 `json:"VatAmount"`
}

// Product represents a NUMIER product.
type Product struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	IDCategory   string  `json:"idCategory"`
	NameCategory string  `json:"nameCategory"`
	NamePrice1   string  `json:"namePrice1"`
	NamePrice2   string  `json:"namePrice2"`
	NamePrice3   string  `json:"namePrice3"`
	NamePrice4   string  `json:"namePrice4"`
	Price1       float64 `json:"price1"`
	Price2       float64 `json:"price2"`
	Price3       float64 `json:"price3"`
	Price4       float64 `json:"price4"`
	VatType      float64 `json:"vatType"`
	IsActive     bool    `json:"isActive"`
}

// ProductWithRecipe represents a NUMIER product with its subproducts (recipe).
type ProductWithRecipe struct {
	IDProduct    string             `json:"idProduct"`
	Name         string             `json:"name"`
	IDCategory   string             `json:"idCategory"`
	NameCategory string             `json:"nameCategory"`
	Subproducts  []RecipeSubproduct `json:"subproducts"`
}

type RecipeSubproduct struct {
	IDSubproduct string  `json:"idSubproduct"`
	Name         string  `json:"name"`
	Parts        float64 `json:"parts"`
	Escandallo   float64 `json:"escandallo"`
	Measurement  string  `json:"measurement"`
	Type         string  `json:"type"`      // "O", "E", "B"
	Variation    float64 `json:"variation"`
}

// Category represents a NUMIER product category.
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Expense represents a purchase/expense from NUMIER.
type Expense struct {
	Date         string        `json:"Date"`
	Reference    string        `json:"Reference"`
	Provider     Provider      `json:"Provider"`
	ExpenseItems []ExpenseItem `json:"ExpenseItems"`
	Type         string        `json:"Type"`    // "Factura" or "Albarán"
	Origin       string        `json:"Origin"`
	Payments     string        `json:"Payments"`
	Totals       SaleTotals    `json:"Totals"`
}

type Provider struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	TaxID string `json:"taxId"`
}

type ExpenseItem struct {
	IDSubproduct string  `json:"idSubproduct"`
	Name         string  `json:"name"`
	Reference    string  `json:"reference"`
	Type         string  `json:"type"` // unit type e.g. "Kgs"
	PriceTag     float64 `json:"priceTag"`
	Units        float64 `json:"units"`
	Parts        float64 `json:"parts"`
	VatType      float64 `json:"vatType"`
	NetPrice     float64 `json:"netPrice"`
	GrossPrice   float64 `json:"grossPrice"`
	Dto          float64 `json:"dto"` // discount
	NetAmount    float64 `json:"netAmount"`
	GrossAmount  float64 `json:"grossAmount"`
}

// Zona represents a POS terminal.
type Zona struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
