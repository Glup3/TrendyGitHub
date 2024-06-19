package jobs

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
)

// 1000 repositories == 1 Unit
const repoCountToRateLimitUnitRatio = 1000
const starCountToRateLimitUnitRatio = 100
const maxRestPageCount = 400
const bufferUnits = 3

func FetchNextRepositoryHistory(db *database.Database, ctx context.Context, githubToken string) {
	dataLoader := loader.NewAPILoader(ctx, githubToken)

	repo, err := FetchNextMissingRepository(db, ctx, githubToken, true)
	if err != nil {
		log.Error().Err(err).Msg("fetching repo failed")
		return
	}

	log.Info().Msgf("fetching star history for repo %s", repo.NameWithOwner)

	FetchStarHistory(db, ctx, dataLoader, *repo)

	RefreshViews(db, ctx)

	log.Print("done fetching missing star histories")
}

func FetchNextMissingRepository(db *database.Database, ctx context.Context, githubToken string, useREST bool) (*database.MissingRepo, error) {
	dataLoader := loader.NewAPILoader(ctx, githubToken)
	var remainingUnits int

	// totalRepoCount, err := database.GetTotalRepoCount(db, ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed fetching total repo count")
	// }

	if useREST {
		rateLimitRest, err := dataLoader.GetRateLimitRest()
		if err != nil {
			return nil, fmt.Errorf("failed fetching rate limit rest %v", err)
		}
		remainingUnits = rateLimitRest.Rate.Remaining
	} else {
		rateLimitGraphql, err := dataLoader.GetRateLimit()
		if err != nil {
			return nil, fmt.Errorf("failed fetching rate limit graphql %v", err)
		}
		remainingUnits = rateLimitGraphql.Remaining
	}

	// reservedUnits := int(math.Floor(float64(totalRepoCount) / repoCountToRateLimitUnitRatio))
	// remainingUnits -= reservedUnits

	if remainingUnits <= 0 {
		return nil, fmt.Errorf("remaining rate limit is not enough (REST %v) - aborting", useREST)
	}

	log.Info().Msgf("rate limit: %d units remaining (REST %v)", remainingUnits, useREST)
	log.Info().Msg("fetching missing star history")

	maxStarCount := remainingUnits*starCountToRateLimitUnitRatio - bufferUnits
	if useREST && maxStarCount > starCountToRateLimitUnitRatio*maxRestPageCount {
		maxStarCount = starCountToRateLimitUnitRatio * maxRestPageCount
	}

	repo, err := database.GetNextMissingHistoryRepo(db, ctx, maxStarCount)
	if err != nil {
		log.Error().Err(err).Msg("failed fetching next missing repo")
	}

	return &repo, nil
}

func FetchStarHistory(db *database.Database, ctx context.Context, dataLoader loader.DataLoader, repo database.MissingRepo) {
	const maxConcurrentRequests = 80

	firstPage := 1
	_, pageInfo, err := dataLoader.LoadRepoStarHistoryPage(repo.NameWithOwner, firstPage)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load first page")
	}

	if pageInfo == nil {
		log.Fatal().Msg("failed to get pagination info from first page")
	}

	totalPages := pageInfo.LastPage
	if totalPages == 0 {
		totalPages = 1
	}

	log.Info().Msgf("total pages: %d", totalPages)

	var wg sync.WaitGroup
	var mu sync.Mutex
	timestamps := make([]time.Time, 0)

	pageCh := make(chan int, totalPages)
	resultCh := make(chan []time.Time, totalPages)
	errCh := make(chan error, totalPages)

	worker := func() {
		defer wg.Done()
		for page := range pageCh {
			pageTimestamps, _, err := dataLoader.LoadRepoStarHistoryPage(repo.NameWithOwner, page)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- pageTimestamps
		}
	}

	for i := 0; i < maxConcurrentRequests; i++ {
		wg.Add(1)
		go worker()
	}

	for page := 1; page <= totalPages; page++ {
		pageCh <- page
	}
	close(pageCh)

	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	for pageTimestamps := range resultCh {
		mu.Lock()
		timestamps = append(timestamps, pageTimestamps...)
		mu.Unlock()
	}

	if len(errCh) > 0 {
		log.Fatal().Err(<-errCh).Msg("error loading star history")
	}

	log.Info().Msgf("total timestamps: %d", len(timestamps))

	starCounts := make(map[time.Time]int)
	cumulativeCounts := make(map[time.Time]int)

	countStars(&starCounts, timestamps)
	calculateCumulativeStars(&cumulativeCounts, starCounts)

	var inputs []database.StarHistoryInput
	for key, value := range cumulativeCounts {
		inputs = append(inputs, database.StarHistoryInput{
			Id:        repo.Id,
			CreatedAt: key,
			StarCount: value,
		})
	}

	err = database.BatchUpsertStarHistory(db, ctx, inputs)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to upsert star history %s", repo.NameWithOwner)
	}

	log.Info().Msgf("finished upserting star history for repo %s", repo.NameWithOwner)
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
