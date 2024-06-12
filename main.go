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
	cursors := []string{
		"Y3Vyc29yOjEwMA==", "Y3Vyc29yOjIwMA==", "Y3Vyc29yOjMwMA==",
		"Y3Vyc29yOjQwMA==", "Y3Vyc29yOjUwMA==", "Y3Vyc29yOjYwMA==",
		"Y3Vyc29yOjcwMA==", "Y3Vyc29yOjgwMA==", "Y3Vyc29yOjkwMA==", "",
	}

	// TODO: make it possible via settings to toggle between file loaders
	if true {
		dataLoader = loader.NewAPILoader(ctx, configs.GitHubToken)
	}

	for i := 0; i < 6; i++ {
		settings, err := database.LoadSettings(db, ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}

		fmt.Println("fetching for max star count", settings.CurrentMaxStarCount)
		repos, pageInfo, err := dataLoader.LoadMultipleRepos(settings.CurrentMaxStarCount, cursors)
		if err != nil {
			// TODO: detecting 403 rate limit errors
			log.Printf("Some fetches failed: %v", err)
		}

		inputs := config.MapGitHubReposToInputs(repos)
		ids, err := database.UpsertRepositories(db, ctx, inputs)
		if err != nil {
			log.Fatalf("%v", err)
		}

		fmt.Println("upserted", len(ids), "repositories")
		fmt.Println("max star count now is", pageInfo.NextMaxStarCount)
		fmt.Println()

		err = database.UpdateCurrentMaxStarCount(db, ctx, settings.ID, pageInfo.NextMaxStarCount)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	fmt.Println("Done")
}
