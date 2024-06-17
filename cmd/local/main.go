package main

import (
	"context"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
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

	var missingRepos []database.MissingRepo
	dataLoader := loader.NewAPILoader(ctx, configs.GitHubToken)

	rateLimit, err := dataLoader.GetRateLimit()
	if err != nil {
		log.Fatal(err)
	}

	if rateLimit.Remaining == 0 {
		log.Print("Rate limit is 0, skipping - next time is", rateLimit.ResetAt)
	}

	maxStarCount := 25_000
	remainingUnits := rateLimit.Remaining
	bufferUnits := 5

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

	// rows, err := database.CreateSnapshotAndReset(db, ctx, 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// log.Println("Created snapshot with repo count", rows)
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
