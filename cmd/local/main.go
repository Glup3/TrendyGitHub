package main

import (
	"context"
	"log"
	"time"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/github"
	"github.com/glup3/TrendyGitHub/internal/jobs"
	"github.com/glup3/TrendyGitHub/internal/repository"
)

func main() {
	ctx := context.Background()

	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
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

	client := github.NewClient(configs.GitHubToken)

	untilDate := time.Date(2024, time.June, 22, 0, 0, 0, 0, time.UTC)
	job := jobs.NewHistoryJob(ctx, db, nil, client)
	err = job.Repair(repository.BrokenRepo{
		Id:            970,
		StarCount:     39957,
		NameWithOwner: "THUDM/ChatGLM-6B",
		UntilDate:     untilDate,
	})

	if err != nil {
		log.Fatalf("why %v", err)
	}
}
