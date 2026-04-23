// probe-iikoweb is a development utility that authenticates against an
// iikoWeb tenant and probes a list of candidate endpoints to help discover
// the data-side API surface (sales, stock, invoices, OLAP-equivalent).
//
// Usage:
//
//	IIKOWEB_URL=https://youcook-ala.iikoweb.ru \
//	IIKOWEB_LOGIN=Alex \
//	IIKOWEB_PASSWORD='real-password' \
//	go run ./cmd/probe-iikoweb/
//
// On success it logs the version, authenticates, hits /api/stores/list, then
// probes each candidate endpoint (GET) and reports HTTP status + first 200 chars
// of the response body. Promote successful endpoints to typed methods on
// internal/iikoweb/api.go.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/iikoweb"
	"github.com/joho/godotenv"
)

// candidateEndpoints lists known + speculative iikoWeb / iikoOffice paths to
// probe under a live session. Add more here as you discover them in the SPA
// chunks or browser network panel.
var candidateEndpoints = []string{
	// Confirmed in navigator SPA chunks
	"/api/stores/list",
	"/api/kpi-metric/stores",
	"/api/permissions/my",
	"/api/brand/get",
	"/api/config/get",
	"/api/wizard/is-needed",
	// Speculative — typical iikoOffice module roots; any non-404 means we've
	// hit a live data module worth grepping further.
	"/api/sales/list",
	"/api/orders/list",
	"/api/products/list",
	"/api/nomenclature/list",
	"/api/balance/list",
	"/api/stock/list",
	"/api/invoices/list",
	"/api/reports/olap",
	"/api/olap/sales",
	"/api/olap/transactions",
	"/api/transactions/list",
	"/api/employees/list",
	"/api/suppliers/list",
}

func main() {
	_ = godotenv.Load()

	url := strings.TrimRight(os.Getenv("IIKOWEB_URL"), "/")
	login := os.Getenv("IIKOWEB_LOGIN")
	password := os.Getenv("IIKOWEB_PASSWORD")
	if url == "" || login == "" || password == "" {
		fatalf("IIKOWEB_URL, IIKOWEB_LOGIN, IIKOWEB_PASSWORD must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := iikoweb.NewClient(url, login, password)
	if err != nil {
		fatalf("create client: %v", err)
	}

	// Pre-flight: tenant info + version
	status, err := client.GetAuthStatus(ctx)
	if err != nil {
		fatalf("status probe: %v", err)
	}
	fmt.Printf("Tenant: %s (%s) — iikoWeb %s — authorized=%v\n",
		status.Domain, status.ClientName, status.AppVersion, status.Authorized)

	if err := client.Authenticate(ctx); err != nil {
		fatalf("authenticate: %v", err)
	}
	fmt.Println("Authenticated OK — session cookie established")

	stores, err := client.GetStores(ctx)
	if err != nil {
		fmt.Printf("[warn] /api/stores/list failed: %v\n", err)
	} else {
		fmt.Printf("Stores: %d\n", len(stores))
		for _, s := range stores {
			fmt.Printf("  - %s | %s\n", s.ID, s.Name)
		}
	}

	fmt.Println()
	fmt.Println("=== Endpoint discovery probe ===")
	for _, path := range candidateEndpoints {
		body, err := client.ProbeEndpoint(ctx, path)
		if err != nil {
			fmt.Printf("  %-40s ERROR: %v\n", path, truncate(err.Error(), 100))
			continue
		}
		fmt.Printf("  %-40s OK   %s\n", path, truncate(string(body), 200))
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "probe-iikoweb: "+format+"\n", args...)
	os.Exit(1)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
