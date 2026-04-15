// Probe iiko Server API write endpoints to see which (if any) accept document submissions.
// We're checking: can we push back inventory recounts and unit-price corrections?
//
// Strategy: GET each candidate URL — 405 (Method Not Allowed) means the endpoint EXISTS
// and accepts only POST; 404 means it doesn't exist at all on this iiko version.
// Then for the surviving 405 candidates, we do a minimal POST to see what payload shape
// they accept (will fail with 400 + a useful error message about missing fields).
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
		fmt.Println("fetch creds:", err)
		os.Exit(1)
	}
	baseURL = strings.TrimRight(baseURL, "/")

	h := sha1.Sum([]byte(password))
	authURL := fmt.Sprintf("%s/resto/api/auth?login=%s&pass=%s",
		baseURL, url.QueryEscape(login), url.QueryEscape(fmt.Sprintf("%x", h)))
	resp, err := http.Get(authURL)
	if err != nil {
		fmt.Println("auth:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	token := strings.TrimSpace(string(body))
	fmt.Println("auth OK, token len:", len(token))

	// Candidates for write endpoints across iiko Server versions.
	candidates := []string{
		"/resto/api/documents/import/inventory",
		"/resto/api/documents/import/incomingInvoice",
		"/resto/api/documents/import/internalTransfer",
		"/resto/api/documents/import/writeoff",
		"/resto/api/documents/inventory",
		"/resto/api/v2/documents/inventory",
		"/resto/api/v2/documents/import/inventory",
		"/resto/api/v2/entities/products/update",
		"/resto/api/products/update",
	}

	probe := func(method, path string, body string) {
		full := fmt.Sprintf("%s%s?key=%s", baseURL, path, url.QueryEscape(token))
		var req *http.Request
		var err error
		if body != "" {
			req, err = http.NewRequest(method, full, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/xml")
		} else {
			req, err = http.NewRequest(method, full, nil)
		}
		if err != nil {
			fmt.Printf("  %s %-50s ERR %v\n", method, path, err)
			return
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("  %s %-50s ERR %v\n", method, path, err)
			return
		}
		rb, _ := io.ReadAll(r.Body)
		r.Body.Close()
		s := strings.ReplaceAll(string(rb), "\n", " ")
		if len(s) > 250 {
			s = s[:250] + "..."
		}
		fmt.Printf("  %s %-50s → %d  %s\n", method, path, r.StatusCode, s)
	}

	fmt.Println("\n=== GET probe (405 = exists, POST-only; 404 = missing):")
	for _, p := range candidates {
		probe("GET", p, "")
	}

	fmt.Println("\n=== POST minimal XML probe to surviving candidates:")
	minimalInvDoc := `<?xml version="1.0" encoding="UTF-8"?><document/>`
	for _, p := range candidates {
		probe("POST", p, minimalInvDoc)
	}

	fmt.Println("\n=== POST JSON probe to /resto/api/v2/entities/products/update:")
	probeJSON := func(path string, body string) {
		full := fmt.Sprintf("%s%s?key=%s", baseURL, path, url.QueryEscape(token))
		req, _ := http.NewRequest("POST", full, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("  POST %s ERR %v\n", path, err)
			return
		}
		rb, _ := io.ReadAll(r.Body)
		r.Body.Close()
		s := strings.ReplaceAll(string(rb), "\n", " ")
		if len(s) > 400 {
			s = s[:400] + "..."
		}
		fmt.Printf("  POST %s → %d  %s\n", path, r.StatusCode, s)
	}
	// Try a few payload shapes
	probeJSON("/resto/api/v2/entities/products/update", `{}`)

	// Now fetch ONE real product with its full ProductDto to see field names.
	fmt.Println("\n=== Fetching a real product to inspect ProductDto schema:")
	listURL := fmt.Sprintf("%s/resto/api/v2/entities/products/list?key=%s", baseURL, url.QueryEscape(token))
	r, err := http.Get(listURL)
	if err != nil {
		fmt.Println("get list:", err)
		return
	}
	listBody, _ := io.ReadAll(r.Body)
	r.Body.Close()
	// Cut to first ~1500 chars of the JSON list to inspect schema
	s := string(listBody)
	if len(s) > 1500 {
		s = s[:1500] + "..."
	}
	fmt.Println("first ~1500 chars of products list:")
	fmt.Println(s)
}
