package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/foodbi/backend/internal/numier"
)

func main() {
	apiKey := os.Getenv("NUMIER_API_KEY")
	if apiKey == "" {
		fmt.Println("NUMIER_API_KEY env var required")
		os.Exit(1)
	}
	tpvID := os.Getenv("NUMIER_TPV_ID")
	if tpvID == "" {
		tpvID = "7186"
	}

	ctx := context.Background()
	client := numier.NewClient(apiKey)

	// 1. Test GetLocales
	fmt.Println("=== GetLocales ===")
	locales, err := client.GetLocales(ctx)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		printJSON(locales)
	}

	// 2. Test GetCategories
	fmt.Printf("\n=== GetCategories (TPV %s) ===\n", tpvID)
	cats, err := client.GetCategories(ctx, tpvID)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		printJSON(cats)
	}

	// 3. Test GetProducts (page 1)
	fmt.Printf("\n=== GetProducts (TPV %s, page 1) ===\n", tpvID)
	products, totalPages, err := client.GetProducts(ctx, tpvID, 1)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Total pages: %d, products on page: %d\n", totalPages, len(products))
		if len(products) > 3 {
			printJSON(products[:3])
		} else {
			printJSON(products)
		}
	}

	// 4. Test GetSales (last 7 days, page 1)
	fmt.Printf("\n=== GetSales (TPV %s, last 7 days, page 1) ===\n", tpvID)
	sales, salesPages, err := client.GetSales(ctx, tpvID, "2026-04-09", "2026-04-16", 1)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Total pages: %d, sales on page: %d\n", salesPages, len(sales))
		if len(sales) > 2 {
			printJSON(sales[:2])
		} else {
			printJSON(sales)
		}
	}

	// 5. Test GetExpenses (last 30 days, page 1)
	fmt.Printf("\n=== GetExpenses (TPV %s, last 30 days, page 1) ===\n", tpvID)
	expenses, expPages, err := client.GetExpenses(ctx, tpvID, "2026-03-17", "2026-04-16", 1)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Total pages: %d, expenses on page: %d\n", expPages, len(expenses))
		if len(expenses) > 2 {
			printJSON(expenses[:2])
		} else {
			printJSON(expenses)
		}
	}

	fmt.Println("\n=== Probe complete ===")
}

func printJSON(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
