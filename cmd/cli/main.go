package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
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
		runDaily(db, ctx, configs.GitHubToken)
	case "history":
		runHistory(db, ctx, configs.GitHubToken)
	default:
		fmt.Printf("Invalid mode: %s. Use 'daily' or 'history'.\n", mode)
		os.Exit(1)
	}
}

func runDaily(db *database.Database, ctx context.Context, githubToken string) {
	unitCount := 0
	cursors := []string{
		"Y3Vyc29yOjEwMA==", "Y3Vyc29yOjIwMA==", "Y3Vyc29yOjMwMA==",
		"Y3Vyc29yOjQwMA==", "Y3Vyc29yOjUwMA==", "Y3Vyc29yOjYwMA==",
		"Y3Vyc29yOjcwMA==", "Y3Vyc29yOjgwMA==", "Y3Vyc29yOjkwMA==", "",
	}

	dataLoader := loader.NewAPILoader(ctx, githubToken)

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

	refreshViews(db, ctx)

	fmt.Println("Done")
}

func runHistory(db *database.Database, ctx context.Context, githubToken string) {
	var missingRepos []database.MissingRepo
	dataLoader := loader.NewAPILoader(ctx, githubToken)

	rateLimit, err := dataLoader.GetRateLimit()
	if err != nil {
		log.Fatal(err)
	}

	if rateLimit.Remaining == 0 {
		log.Print("Rate limit is 0, skipping - next time is", rateLimit.ResetAt)
	} else {
		log.Print("running job - fetching missing star histories")
	}

	maxStarCount := 25_000
	remainingUnits := rateLimit.Remaining
	bufferUnits := 1

	for {
		excludedIds := make([]int32, len(missingRepos))
		for i, repo := range missingRepos {
			excludedIds[i] = repo.Id
		}

		repo, err := database.GetNextMissingHistoryRepo(db, ctx, maxStarCount, excludedIds)
		if err != nil {
			log.Println(err)
			break
		}

		estimatedUnits := int(math.Ceil(float64(repo.StarCount)/100)) + bufferUnits

		if estimatedUnits > remainingUnits {
			break
		}

		remainingUnits -= estimatedUnits
		missingRepos = append(missingRepos, repo)
	}

	var wg sync.WaitGroup
	const maxConcurrency = 3

	semaphore := make(chan struct{}, maxConcurrency)

	for _, repo := range missingRepos {
		wg.Add(1)

		semaphore <- struct{}{}

		go func(repo database.MissingRepo) {
			defer wg.Done()
			defer func() { <-semaphore }()

			loadStarHistory(db, ctx, dataLoader, repo)
		}(repo)
	}

	wg.Wait()

	refreshViews(db, ctx)

	log.Print("done fetching missing star histories")
}

func loadStarHistory(db *database.Database, ctx context.Context, dataLoader loader.DataLoader, repo database.MissingRepo) {
	cursor := ""
	var totalDates []time.Time
	pageCounter := 0

	for {
		dates, info, err := dataLoader.LoadRepoStarHistoryDates(repo.GithubId, cursor)
		if err != nil {
			log.Print("aborting loading star history!", repo.GithubId, pageCounter, err)
		}

		cursor = info.NextCursor
		pageCounter++
		totalDates = append(totalDates, dates...)

		if pageCounter%10 == 0 {
			log.Printf("githubId: %s - loaded page %d of %d page", repo.GithubId, pageCounter, info.TotalStars/100)
		}

		if !info.HasNextPage {
			break
		}
	}

	log.Println("finished loading, total length", len(totalDates))

	starCounts := make(map[time.Time]int)
	cumulativeCounts := make(map[time.Time]int)

	countStars(&starCounts, totalDates)
	calculateCumulativeStars(&cumulativeCounts, starCounts)

	var inputs []database.StarHistoryInput
	for key, value := range cumulativeCounts {
		inputs = append(inputs, database.StarHistoryInput{
			Id:        repo.Id,
			CreatedAt: key,
			StarCount: value,
		})
	}

	err := database.BatchUpsertStarHistory(db, ctx, inputs)
	if err != nil {
		log.Fatal("failed to upsert star history", err)
	}

	log.Printf("finished upserting star history for repo id %d", repo.Id)
}

// normalizeDate normalizes a time.Time to midnight of the same day
func normalizeDate(t time.Time) time.Time {
	return t.Truncate(24 * time.Hour)
}

func countStars(starCounts *map[time.Time]int, dateTimes []time.Time) {
	for _, dateTime := range dateTimes {
		normalizedDate := normalizeDate(dateTime)
		(*starCounts)[normalizedDate]++
	}
}

func calculateCumulativeStars(cumulativeCounts *map[time.Time]int, starCounts map[time.Time]int) {
	var keys []time.Time
	for date := range starCounts {
		keys = append(keys, date)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	cumulativeSum := 0
	for _, key := range keys {
		cumulativeSum += starCounts[key]
		(*cumulativeCounts)[key] = cumulativeSum
	}
}

func refreshViews(db *database.Database, ctx context.Context) {
	log.Print("refreshing views...")

	var errors []error

	err := database.RefreshHistoryView(db, ctx, "mv_daily_stars")
	if err != nil {
		errors = append(errors, err)
	}

	err = database.RefreshHistoryView(db, ctx, "mv_weekly_stars")
	if err != nil {
		errors = append(errors, err)
	}

	err = database.RefreshHistoryView(db, ctx, "mv_monthly_stars")
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		log.Println(errors)
	}
}
