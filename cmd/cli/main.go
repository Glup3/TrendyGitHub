package main

import (
	"context"
	"fmt"
	"log"
	"os"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/jobs"
)

func main() {
	ctx := context.Background()

	if len(os.Args) < 2 {
		log.Println("Usage: ./tgh [daily|history]")
		os.Exit(1)
	}

	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	db, err := database.NewDatabase(ctx, configs.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	mode := os.Args[1]

	switch mode {
	case "daily":
		jobs.RunDaily(db, ctx, configs.GitHubToken)
	case "history":
		jobs.RunHistory(db, ctx, configs.GitHubToken)
	default:
		fmt.Printf("Invalid mode: %s. Use 'daily' or 'history'.\n", mode)
		os.Exit(1)
	}
}
