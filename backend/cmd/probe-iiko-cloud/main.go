package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/foodbi/backend/internal/iikocloud"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	apiLogin := os.Getenv("IIKO_CLOUD_API_LOGIN")
	if apiLogin == "" {
		fmt.Fprintln(os.Stderr, "IIKO_CLOUD_API_LOGIN env var is required")
		os.Exit(1)
	}

	ctx := context.Background()
	client := iikocloud.NewClient(apiLogin)

	fmt.Printf("Authenticating with iiko Cloud (api-ru.iiko.services) as %q ...\n", apiLogin)
	if err := client.Authenticate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Auth failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Auth OK — token obtained.")

	fmt.Println("\nFetching organizations...")
	orgs, err := client.GetOrganizations(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetOrganizations failed: %v\n", err)
		os.Exit(1)
	}

	if len(orgs) == 0 {
		fmt.Println("No organizations found for this apiLogin.")
		return
	}

	fmt.Printf("Found %d organization(s):\n", len(orgs))
	for _, org := range orgs {
		fmt.Printf("  ID: %s  Name: %s\n", org.ID, org.Name)
	}

	// Print full JSON for reference.
	b, _ := json.MarshalIndent(orgs, "  ", "  ")
	fmt.Printf("\nFull JSON:\n  %s\n", string(b))

	// Also probe terminal groups for the first org.
	if len(orgs) > 0 {
		fmt.Printf("\nFetching terminal groups for org %s...\n", orgs[0].ID)
		tgs, err := client.GetTerminalGroups(ctx, []string{orgs[0].ID})
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetTerminalGroups failed: %v\n", err)
		} else {
			b2, _ := json.MarshalIndent(tgs, "  ", "  ")
			fmt.Printf("Terminal groups:\n  %s\n", string(b2))
		}
	}
}
