package jobs

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/glup3/TrendyGitHub/internal/repository"
)

// 100 repositories == 1 Unit
const (
	repoCountToRateLimitUnitRatio = 100
)

// GitHub REST API limitation: maximum pagination of 400 pages
func (job *HistoryJob) FetchHistoryUnder40kStars() {
	const maxAPILimitStarCount = 40_000
	const maxAPILimitPages = 400
	updatedCount := 0

	for {
		rateLimit, err := (*job.loader).GetRateLimitRest()
		if err != nil {
			log.Error().Err(err).Msg("failed fetching rate limit REST")
			break
		}

		if rateLimit.Rate.Remaining <= 0 {
			log.Warn().Int("resetAt", rateLimit.Rate.Reset).Msg("REST API rate limit exceeded")
			break
		}

		maxStarCount := rateLimit.Rate.Remaining * 400
		if maxStarCount > maxAPILimitStarCount {
			maxStarCount = maxAPILimitStarCount
		}

		repo, err := job.repoRepository.FindNextMissing(maxStarCount, repository.OrderAsc)
		if err != nil {
			log.Warn().
				Err(err).
				Int("maxStarCount", maxStarCount).
				Int("remainingLimit", rateLimit.Rate.Remaining).
				Msg("failed fetching next missing repo REST")
			break
		}

		log.Info().
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Str("githubId", repo.GithubId).
			Int("remainingLimit", rateLimit.Rate.Remaining).
			Msg("fetching history for repo REST")

		err = job.FetchStarHistory(repo)
		if err != nil {
			log.Error().
				Err(err).
				Int("id", repo.Id).
				Str("repository", repo.NameWithOwner).
				Msg("something happend when fetching REST API star history")
			break
		}

		updatedCount++
	}

	log.Info().Int("count", updatedCount).Msg("REST: done fetching missing star histories")
}

func (job *HistoryJob) FetchHistory() {
	updatedCount := 0

	for {
		rateLimit, err := (*job.loader).GetRateLimit()
		if err != nil {
			log.Error().Err(err).Msg("failed fetching rate limit GraphQL")
			break
		}

		if rateLimit.Remaining <= 0 {
			log.Warn().Time("resetAt", rateLimit.ResetAt).Msg("GraphQL rate limit exceeded")
			break
		}

		maxStarCount := rateLimit.Remaining * 100
		repo, err := job.repoRepository.FindNextMissing(maxStarCount, repository.OrderDesc)
		if err != nil {
			log.Warn().
				Err(err).
				Int("maxStarCount", maxStarCount).
				Int("remainingLimit", rateLimit.Remaining).
				Msg("failed fetching next missing repo GraphQL")
			break
		}

		log.Info().
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Int("remainingLimit", rateLimit.Remaining).
			Msg("fetching history for repo GraphQL")

		cursor := ""
		var totalDates []time.Time
		pageCounter := 0

		for {
			dates, info, err := (*job.loader).LoadRepoStarHistoryDates(repo.GithubId, cursor)
			if err != nil {
				if strings.Contains(err.Error(), "Could not resolve to a node") ||
					strings.Contains(err.Error(), "Unavailable For Legal Reasons") {
					log.Warn().
						Err(err).
						Str("repository", repo.NameWithOwner).
						Int("id", repo.Id).
						Msg("deleting repo because it doesn't exist anymore")

					err = job.deleteDeadRepo(repo)
					if err != nil {
						log.Fatal().
							Err(err).
							Int("id", repo.Id).
							Str("repository", repo.NameWithOwner).
							Msg("failed to delete dead repo")
					}

					break
				} else {
					log.Fatal().
						Err(err).
						Int("id", repo.Id).
						Str("repository", repo.NameWithOwner).
						Msg("aborting loading star history GraphQL")
				}
			}

			cursor = info.NextCursor
			pageCounter++
			totalDates = append(totalDates, dates...)

			if pageCounter%10 == 0 {
				log.Info().
					Int("id", repo.Id).
					Str("repository", repo.NameWithOwner).
					Int("page", pageCounter).
					Int("totalPages", info.TotalStars/100).
					Msg("fetched page")
			}

			if !info.HasNextPage {
				break
			}
		}

		err = job.aggregateAndInsertHistory(totalDates, repo)
		if err != nil {
			log.Error().
				Err(err).
				Str("githubId", repo.GithubId).
				Str("repository", repo.NameWithOwner).
				Msg("something happend when aggregating star history")
			break
		}

		updatedCount++
	}

	log.Info().Int("count", updatedCount).Msg("GraphQL: done fetching missing star histories")
}

func (job *HistoryJob) FetchStarHistory(repo repository.Repo) error {
	timestamps := make([]time.Time, 0)

	page1Timestamps, pageInfo, err := (*job.loader).LoadRepoStarHistoryPage(repo.NameWithOwner, 1)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "451") {
			log.Warn().
				Err(err).
				Str("repository", repo.NameWithOwner).
				Int("id", repo.Id).
				Msg("deleting repo because it doesn't exist anymore")
			return job.deleteDeadRepo(repo)
		}

		log.Error().
			Err(err).
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msg("failed to load first page")
		return err
	}

	timestamps = append(timestamps, page1Timestamps...)

	totalPages := pageInfo.LastPage
	if totalPages == 0 {
		return job.aggregateAndInsertHistory(timestamps, repo)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	pageCh := make(chan int, totalPages)
	resultCh := make(chan []time.Time, totalPages)
	errCh := make(chan error, totalPages)

	worker := func() {
		defer wg.Done()
		for page := range pageCh {
			pageTimestamps, _, err := (*job.loader).LoadRepoStarHistoryPage(repo.NameWithOwner, page)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- pageTimestamps
		}
	}

	const maxConcurrentRequests = 80
	for i := 0; i < maxConcurrentRequests; i++ {
		wg.Add(1)
		go worker()
	}

	for page := 2; page <= totalPages; page++ {
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
		err := <-errCh
		log.Error().
			Err(err).
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msg("error loading star history")
		return err
	}

	return job.aggregateAndInsertHistory(timestamps, repo)
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

func (job *HistoryJob) aggregateAndInsertHistory(timestamps []time.Time, repo repository.Repo) error {
	if len(timestamps) == 0 {
		log.Warn().
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msg("no timestamps found, will mark as DONE")

		err := job.repoRepository.MarkAsDone(repo.Id)
		if err != nil {
			log.Error().
				Err(err).
				Int("id", repo.Id).
				Str("repository", repo.NameWithOwner).
				Msg("unable to mark repo as DONE")
			return err
		}

		return nil
	}

	log.Info().
		Int("id", repo.Id).
		Str("repository", repo.NameWithOwner).
		Int("timestamps", len(timestamps)).
		Msgf("aggregating star history")

	starCounts := make(map[time.Time]int)
	cumulativeCounts := make(map[time.Time]int)

	countStars(&starCounts, timestamps)
	calculateCumulativeStars(&cumulativeCounts, starCounts)

	var inputs []repository.StarHistoryInput
	for key, value := range cumulativeCounts {
		inputs = append(inputs, repository.StarHistoryInput{
			Id:        repo.Id,
			CreatedAt: key,
			StarCount: value,
		})
	}

	err := job.historyRepository.BatchUpsertStarHistory(inputs)
	if err != nil {
		log.Error().
			Err(err).
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msgf("failed to upsert star history %s", repo.NameWithOwner)
		return err
	}

	log.Info().
		Int("id", repo.Id).
		Str("repository", repo.NameWithOwner).
		Msgf("finished upserting star history for repo %s", repo.NameWithOwner)

	return nil
}

func (job *HistoryJob) deleteDeadRepo(repo repository.Repo) error {
	err := job.historyRepository.DeleteForRepo(repo.Id)
	if err != nil {
		log.Error().
			Err(err).
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msg("failed to delete dead repo star history")
		return err
	}

	err = job.repoRepository.Delete(repo.Id)
	if err != nil {
		log.Error().
			Err(err).
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msg("failed to delete dead repo")
		return err
	}

	return nil
}
