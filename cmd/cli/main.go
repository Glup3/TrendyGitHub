package main

import (
	"context"
	"fmt"
	"log"
	"time"

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

	unitCount := 0
	cursors := []string{
		"Y3Vyc29yOjEwMA==", "Y3Vyc29yOjIwMA==", "Y3Vyc29yOjMwMA==",
		"Y3Vyc29yOjQwMA==", "Y3Vyc29yOjUwMA==", "Y3Vyc29yOjYwMA==",
		"Y3Vyc29yOjcwMA==", "Y3Vyc29yOjgwMA==", "Y3Vyc29yOjkwMA==", "",
	}

	dataLoader := loader.NewAPILoader(ctx, configs.GitHubToken)

	for {
		settings, err := database.LoadSettings(db, ctx)
		if err != nil {
			log.Fatalf("%v", err)
		}

		if !settings.IsEnabled {
			fmt.Println("Data loading is disabled")
			break
		}

		if settings.CurrentMaxStarCount <= settings.MinStarCount {
			fmt.Println("Reached the end - no more data loading")
			break
		}

		if unitCount >= settings.TimeoutMaxUnits {
			fmt.Printf("Prevent secondary time limit, waiting %d seconds\n", settings.TimeoutSecondsPrevent)
			time.Sleep(time.Duration(settings.TimeoutSecondsPrevent) * time.Second)
			unitCount = 0
		}

		fmt.Println("fetching for max star count", settings.CurrentMaxStarCount)
		repos, pageInfo, err := dataLoader.LoadMultipleRepos(settings.CurrentMaxStarCount, cursors)
		hasLoadError := false
		if err != nil {
			// TODO: detecting 403 rate limit errors
			log.Printf("Some fetches failed: %v", err)
			hasLoadError = true
		}
		unitCount += pageInfo.UnitCosts

		inputs := config.MapGitHubReposToInputs(repos)
		ids, err := database.UpsertRepositories(db, ctx, inputs)
		if err != nil {
			log.Fatalf("%v", err)
		}

		fmt.Println("upserted", len(ids), "repositories")

		if hasLoadError {
			fmt.Printf("Encountered errors! Timeout of %d seconds\n", settings.TimeoutSecondsExceeded)
			time.Sleep(time.Duration(settings.TimeoutSecondsExceeded) * time.Second)
			unitCount = 0
			continue
		}

		if settings.CurrentMaxStarCount == pageInfo.NextMaxStarCount {
			fmt.Println("WARNING: there are more than 1000 repos in this star range - aborting", pageInfo.NextMaxStarCount)
			break
		}

		fmt.Println("max star count now is", pageInfo.NextMaxStarCount)

		err = database.UpdateCurrentMaxStarCount(db, ctx, settings.ID, pageInfo.NextMaxStarCount)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	fmt.Println("Done")
}
