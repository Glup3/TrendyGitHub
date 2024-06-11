package main

import (
	"context"
	"fmt"
	"log"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
)

func main() {
	ctx := context.Background()

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

	var dataLoader loader.DataLoader
	if true {
		dataLoader = loader.NewAPILoader(ctx, configs.GitHubToken)
	}

	// TODO: also load max star count from db
	maxStarCount := 6001 // count: 9

	for i := 0; i < 10; i++ {
		settings, err := database.LoadSettings(db, ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}

		fmt.Println("fetching cursor", settings.CursorValue)
		repos, pageInfo, err := dataLoader.LoadRepos(settings.CursorValue, maxStarCount)
		if err != nil {
			log.Fatalf("fetching repositories failed: %v", err)
		}

		inputs := config.MapGitHubReposToInputs(repos)
		_, err = database.UpsertRepositories(db, ctx, inputs)
		if err != nil {
			log.Fatalf("%v", err)
		}

		err = database.UpdateCursor(db, ctx, settings.ID, pageInfo.NextCursor)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	fmt.Println("Done")
}
