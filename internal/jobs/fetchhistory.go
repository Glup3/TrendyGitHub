package jobs

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
)

// 100 repositories == 1 Unit
const (
	repoCountToRateLimitUnitRatio = 100
)

// GitHub REST API limitation: maximum pagination of 400 pages
func FetchHistoryUnder40kStars(db *database.Database, ctx context.Context, githubToken string) {
	const maxAPILimitStarCount = 40_000
	const maxAPILimitPages = 400
	dataLoader := loader.NewAPILoader(ctx, githubToken)

	for {
		rateLimit, err := dataLoader.GetRateLimitRest()
		if err != nil {
			log.Error().Err(err).Msg("failed fetching REST API rate limit")
			break
		}

		if rateLimit.Rate.Remaining <= 0 {
			log.Warn().Int("resetAt", rateLimit.Rate.Reset).Msg("rate limit has been already exhausted")
			break
		}

		maxStarCount := rateLimit.Rate.Remaining * 400
		if maxStarCount > maxAPILimitStarCount {
			maxStarCount = maxAPILimitStarCount
		}

		repo, err := database.GetNextMissingHistoryRepo(db, ctx, maxStarCount, true)
		if err != nil {
			log.Fatal().Err(err).Msg("failed fetching next missing repo")
		}

		log.Info().
			Str("repository", repo.NameWithOwner).
			Str("githubId", repo.GithubId).
			Int32("id", repo.Id).
			Int("remainingLimit", rateLimit.Rate.Remaining).
			Msg("fetching history for repo")

		FetchStarHistory(db, ctx, dataLoader, repo)
	}

	RefreshViews(db, ctx)

	log.Info().Msg("done fetching missing star histories")
}

func FetchHistory(db *database.Database, ctx context.Context, githubToken string) {
	dataLoader := loader.NewAPILoader(ctx, githubToken)

	for {
		rateLimit, err := dataLoader.GetRateLimit()
		if err != nil {
			log.Error().Err(err).Msg("failed fetching GraphQL API rate limit")
			break
		}

		if rateLimit.Remaining <= 0 {
			log.Error().Err(err).Msg("rate limit has been already exhausted")
			break
		}

		maxStarCount := rateLimit.Remaining * 100
		repo, err := database.GetNextMissingHistoryRepo(db, ctx, maxStarCount, false)
		if err != nil {
			log.Error().Err(err).Msg("failed fetching next missing repo")
			break
		}

		log.Info().
			Str("repository", repo.NameWithOwner).
			Str("githubId", repo.GithubId).
			Int32("id", repo.Id).
			Int("remainingLimit", rateLimit.Remaining).
			Msg("fetching history for repo")

		cursor := ""
		var totalDates []time.Time
		pageCounter := 0

		for {
			dates, info, err := dataLoader.LoadRepoStarHistoryDates(repo.GithubId, cursor)
			if err != nil {
				log.Fatal().Err(err).Msg("aborting loading star history!")
			}

			cursor = info.NextCursor
			pageCounter++
			totalDates = append(totalDates, dates...)

			if pageCounter%10 == 0 {
				log.Info().
					Str("githubId", repo.GithubId).
					Str("repository", repo.NameWithOwner).
					Int("page", pageCounter).
					Int("totalPages", info.TotalStars/100).
					Msg("fetched page")
			}

			if !info.HasNextPage {
				break
			}
		}

		AggregateAndInsertHistory(db, ctx, totalDates, repo)
	}

	RefreshViews(db, ctx)
	log.Info().Msg("done fetching missing star histories")
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
	AggregateAndInsertHistory(db, ctx, timestamps, repo)
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

// TODO: refactor struct to general repo & not missing repo
func AggregateAndInsertHistory(db *database.Database, ctx context.Context, timestamps []time.Time, repo database.MissingRepo) {
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

	err := database.BatchUpsertStarHistory(db, ctx, inputs)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to upsert star history %s", repo.NameWithOwner)
	}

	log.Info().Msgf("finished upserting star history for repo %s", repo.NameWithOwner)
}
