package main

import (
	"context"
	"log"
	"sort"
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

	dataLoader := loader.NewAPILoader(ctx, configs.GitHubToken)

	ids, err := database.GetNextMissingHistoryIds(db, ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, id := range ids {
		loadStarHistory(db, ctx, dataLoader, id)
	}

	// rows, err := database.CreateSnapshotAndReset(db, ctx, 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// log.Println("Created snapshot with repo count", rows)
}

func loadStarHistory(db *database.Database, ctx context.Context, dataLoader loader.DataLoader, repoId int32) {
	githubId, err := database.GetGitHubId(db, ctx, repoId)
	if err != nil {
		log.Println(err)
	}

	cursor := ""
	var totalDates []time.Time
	var remainingLimit int

	for {
		dates, info, err := dataLoader.LoadRepoStarHistoryDates(githubId, cursor)
		if err != nil {
			log.Println(cursor, err)
			time.Sleep(20 * time.Second)
			continue
		}

		cursor = info.NextCursor
		totalDates = append(totalDates, dates...)
		remainingLimit = info.RateLimitRemaining

		log.Println("loaded page", remainingLimit, cursor)

		if cursor == "" && info.TotalStars/100 > remainingLimit {
			log.Fatal("not enough remaining limit points - next reset is at", remainingLimit)
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
			Id:        int32(repoId),
			CreatedAt: key,
			StarCount: value,
		})
	}

	err = database.BatchUpsertStarHistory(db, ctx, inputs)
	if err != nil {
		log.Fatal("failed to upsert star history", err)
	}

	log.Printf("finished upserting star history for repo id %d - remaining limit: %d", repoId, remainingLimit)
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
