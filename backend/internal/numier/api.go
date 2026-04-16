package numier

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// GetLocales returns businesses (establishments) linked to the API key.
func (c *Client) GetLocales(ctx context.Context) ([]Locale, error) {
	data, err := c.doRequest(ctx, "/getLocales", nil)
	if err != nil {
		return nil, fmt.Errorf("get locales: %w", err)
	}
	result, _, err := parseResponse[[]Locale](data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetSales returns sales tickets for a specific TPV (POS terminal) between dates.
// Max 250 tickets per page, max 34-day date range.
func (c *Client) GetSales(ctx context.Context, idTpv, startDate, endDate string, page int) ([]Sale, int, error) {
	headers := map[string]string{
		"start_date": startDate,
		"end_date":   endDate,
		"pag":        fmt.Sprintf("%d", page),
	}
	data, err := c.doRequest(ctx, fmt.Sprintf("/v2/sales/%s", idTpv), headers)
	if err != nil {
		return nil, 0, fmt.Errorf("get sales: %w", err)
	}
	result, totalPages, err := parseResponse[[]Sale](data)
	if err != nil {
		return nil, 0, err
	}
	return result, totalPages, nil
}

// GetAllSales fetches all sales for a TPV across all pages within a date range.
func (c *Client) GetAllSales(ctx context.Context, idTpv, startDate, endDate string) ([]Sale, error) {
	var allSales []Sale
	page := 1

	for {
		sales, totalPages, err := c.GetSales(ctx, idTpv, startDate, endDate, page)
		if err != nil {
			return allSales, fmt.Errorf("get sales page %d: %w", page, err)
		}
		allSales = append(allSales, sales...)

		log.Debug().Int("page", page).Int("total_pages", totalPages).Int("sales_on_page", len(sales)).
			Msg("numier: fetched sales page")

		if page >= totalPages || len(sales) == 0 {
			break
		}
		page++
	}

	return allSales, nil
}

// GetProducts returns products for a specific TPV. Max 50 per page.
func (c *Client) GetProducts(ctx context.Context, idTpv string, page int) ([]Product, int, error) {
	headers := map[string]string{
		"pag": fmt.Sprintf("%d", page),
	}
	data, err := c.doRequest(ctx, fmt.Sprintf("/getProducts/%s", idTpv), headers)
	if err != nil {
		return nil, 0, fmt.Errorf("get products: %w", err)
	}
	result, totalPages, err := parseResponse[[]Product](data)
	if err != nil {
		return nil, 0, err
	}
	return result, totalPages, nil
}

// GetAllProducts fetches all products for a TPV across all pages.
func (c *Client) GetAllProducts(ctx context.Context, idTpv string) ([]Product, error) {
	var allProducts []Product
	page := 1

	for {
		products, totalPages, err := c.GetProducts(ctx, idTpv, page)
		if err != nil {
			return allProducts, fmt.Errorf("get products page %d: %w", page, err)
		}
		allProducts = append(allProducts, products...)

		if page >= totalPages || len(products) == 0 {
			break
		}
		page++
	}

	return allProducts, nil
}

// GetProductsWithSubproducts returns products with their recipes. Max 50 per page.
func (c *Client) GetProductsWithSubproducts(ctx context.Context, idTpv string, page int) ([]ProductWithRecipe, int, error) {
	headers := map[string]string{
		"pag": fmt.Sprintf("%d", page),
	}
	data, err := c.doRequest(ctx, fmt.Sprintf("/getProductsWithSubproducts/%s", idTpv), headers)
	if err != nil {
		return nil, 0, fmt.Errorf("get products with subproducts: %w", err)
	}
	result, totalPages, err := parseResponse[[]ProductWithRecipe](data)
	if err != nil {
		return nil, 0, err
	}
	return result, totalPages, nil
}

// GetAllProductsWithSubproducts fetches all products with recipes across all pages.
func (c *Client) GetAllProductsWithSubproducts(ctx context.Context, idTpv string) ([]ProductWithRecipe, error) {
	var all []ProductWithRecipe
	page := 1

	for {
		products, totalPages, err := c.GetProductsWithSubproducts(ctx, idTpv, page)
		if err != nil {
			return all, fmt.Errorf("get recipes page %d: %w", page, err)
		}
		all = append(all, products...)

		if page >= totalPages || len(products) == 0 {
			break
		}
		page++
	}

	return all, nil
}

// GetCategories returns all categories for a TPV. No pagination.
func (c *Client) GetCategories(ctx context.Context, idTpv string) ([]Category, error) {
	data, err := c.doRequest(ctx, fmt.Sprintf("/getCategoriesByTpv/%s", idTpv), nil)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	result, _, err := parseResponse[[]Category](data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetExpenses returns purchases/expenses for a TPV between dates.
func (c *Client) GetExpenses(ctx context.Context, idTpv, startDate, endDate string, page int) ([]Expense, int, error) {
	headers := map[string]string{
		"start_date": startDate,
		"end_date":   endDate,
		"pag":        fmt.Sprintf("%d", page),
	}
	data, err := c.doRequest(ctx, fmt.Sprintf("/v2/expenses/%s", idTpv), headers)
	if err != nil {
		return nil, 0, fmt.Errorf("get expenses: %w", err)
	}
	result, totalPages, err := parseResponse[[]Expense](data)
	if err != nil {
		return nil, 0, err
	}
	return result, totalPages, nil
}

// GetAllExpenses fetches all expenses for a TPV across all pages.
func (c *Client) GetAllExpenses(ctx context.Context, idTpv, startDate, endDate string) ([]Expense, error) {
	var all []Expense
	page := 1

	for {
		expenses, totalPages, err := c.GetExpenses(ctx, idTpv, startDate, endDate, page)
		if err != nil {
			return all, fmt.Errorf("get expenses page %d: %w", page, err)
		}
		all = append(all, expenses...)

		if page >= totalPages || len(expenses) == 0 {
			break
		}
		page++
	}

	return all, nil
}

// GetZonas returns POS terminals related to a given TPV.
func (c *Client) GetZonas(ctx context.Context, idTpv string) ([]Zona, error) {
	data, err := c.doRequest(ctx, fmt.Sprintf("/getZonas/%s", idTpv), nil)
	if err != nil {
		return nil, fmt.Errorf("get zonas: %w", err)
	}
	// Zonas returns nested array: [[{id, name}, ...]]
	result, _, err := parseResponse[[][]Zona](data)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return nil, nil
}
