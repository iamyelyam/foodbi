// Probe iiko assembly chart endpoints to find which one (if any) works for this restaurant.
// Usage: cd backend && go run ./cmd/probe-recipes
package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/database"
	"github.com/foodbi/backend/internal/iiko"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.NewPool(ctx)
	if err != nil {
		fmt.Println("db connect:", err)
		os.Exit(1)
	}
	defer db.Close()

	var baseURL, login, password string
	err = db.QueryRow(ctx,
		`SELECT iiko_server_url, iiko_login, iiko_password FROM companies
		 WHERE iiko_server_url IS NOT NULL AND iiko_server_url <> '' LIMIT 1`).
		Scan(&baseURL, &login, &password)
	if err != nil {
		fmt.Println("fetch company creds:", err)
		os.Exit(1)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	fmt.Println("iiko base URL:", baseURL)

	// Authenticate (replicate iiko.Client logic to get raw token)
	h := sha1.Sum([]byte(password))
	passSHA1 := fmt.Sprintf("%x", h)
	authURL := fmt.Sprintf("%s/resto/api/auth?login=%s&pass=%s",
		baseURL, url.QueryEscape(login), url.QueryEscape(passSHA1))
	resp, err := http.Get(authURL)
	if err != nil {
		fmt.Println("auth request:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	token := strings.TrimSpace(string(body))
	if resp.StatusCode != 200 || token == "" {
		fmt.Printf("auth failed (status %d): %s\n", resp.StatusCode, token)
		os.Exit(1)
	}
	fmt.Println("authenticated, token len:", len(token))

	// Pull nomenclature via the existing client to find a real DISH UUID
	client := iiko.NewClient(baseURL, login, password)
	nomen, err := client.GetNomenclature(ctx)
	if err != nil {
		fmt.Println("get nomenclature:", err)
		os.Exit(1)
	}
	fmt.Println("nomenclature size:", len(nomen))

	// Find dishes (sellable items) — iiko types: DISH, MODIFIER, GOODS, PREPARED
	// We want DISH (готовое блюдо) which has an assembly chart.
	var dishes []iiko.ProductInfo
	typeCount := map[string]int{}
	for _, p := range nomen {
		typeCount[p.Type]++
		if strings.EqualFold(p.Type, "DISH") {
			dishes = append(dishes, p)
		}
	}
	fmt.Println("type breakdown:", typeCount)
	fmt.Println("dishes (type=DISH):", len(dishes))
	if len(dishes) == 0 {
		fmt.Println("no DISH-type products found — falling back to any product with non-empty name")
		for _, p := range nomen {
			if p.Name != "" {
				dishes = append(dishes, p)
				if len(dishes) >= 3 {
					break
				}
			}
		}
	}

	// Probe up to 3 dishes
	maxProbe := 3
	if len(dishes) < maxProbe {
		maxProbe = len(dishes)
	}
	today := time.Now().Format("2006-01-02")
	endpoints := []string{
		"/resto/api/v2/assemblyCharts/getPrepared?productId=%s&date=" + today,
		"/resto/api/v2/assemblyCharts/getPrepared?productId=%s&date=" + today + "T00:00:00",
		"/resto/api/v2/assemblyCharts/getRequired?productId=%s&date=" + today,
		"/resto/api/v2/assemblyCharts/getAssembled?productId=%s&date=" + today,
	}
	_ = time.Now().Format("2006-01-02T15:04:05")

	for _, dish := range dishes[:maxProbe] {
		fmt.Println("\n=== Probing dish:", dish.Name, "(id="+dish.ID+")")

		for _, ep := range endpoints {
			path := fmt.Sprintf(ep, dish.ID)

			sep := "&"
			if !strings.Contains(path, "?") {
				sep = "?"
			}
			full := fmt.Sprintf("%s%s%skey=%s", baseURL, path, sep, url.QueryEscape(token))
			r, err := http.Get(full)
			if err != nil {
				fmt.Printf("  %-60s → ERROR %v\n", path, err)
				continue
			}
			rb, _ := io.ReadAll(r.Body)
			r.Body.Close()
			snippet := string(rb)
			if len(snippet) > 8000 {
				snippet = snippet[:8000] + "..."
			}
			snippet = strings.ReplaceAll(snippet, "\n", " ")
			fmt.Printf("  %-60s → %d  %s\n", path, r.StatusCode, snippet)
		}
	}
}
