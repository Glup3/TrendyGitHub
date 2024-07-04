package jobs

import (
	"context"
	"sort"
	"time"

	database "github.com/glup3/TrendyGitHub/internal/db"
	lo "github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/glup3/TrendyGitHub/internal/repository"
	"github.com/rs/zerolog/log"
)

type HistoryJob struct {
	loader            *lo.Loader
	repoRepository    *repository.RepoRepository
	historyRepository *repository.HistoryRepository
}

func NewHistoryJob(ctx context.Context, db *database.Database, dataLoader *lo.Loader) *HistoryJob {
	return &HistoryJob{
		loader:            dataLoader,
		historyRepository: repository.NewHistoryRepository(ctx, db),
		repoRepository:    repository.NewRepoRepository(ctx, db),
	}
}

func (j *HistoryJob) CreateSnapshot() {
	log.Info().Msg("creating snapshot")

	err := j.historyRepository.CreateSnapshot()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed creating snapshot")
	}

	log.Info().Msg("finished creating snapshot")
}

func (j *HistoryJob) RefreshViews() {
	start := time.Now()
	views := []string{"mv_daily_stars", "mv_weekly_stars", "mv_monthly_stars"}

	log.Info().Msg("refreshing views")

	for _, view := range views {

		start := time.Now()

		err := j.historyRepository.RefreshView(view)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to refresh view %s", view)
		}

		elapsed := time.Since(start)
		log.Info().Msgf("refresh daily view took %s", elapsed)
	}

	log.Info().Msgf("refreshing views took %s", time.Since(start))
}

func (job *HistoryJob) RepairHistory(untilDate time.Time) {

	repos, err := job.repoRepository.GetAllPresentHistoryRepos()
	if err != nil {
		log.Fatal().Err(err).Msg("failed fetching repos")
	}

	if len(repos) == 0 {
		log.Info().Msg("no repos to repair history for")
	}

	for _, repo := range repos {
		rateLimit, err := (*job.loader).GetRateLimit()
		if err != nil {
			log.Error().Err(err).Msg("failed fetching GraphQL API rate limit")
			break
		}

		if rateLimit.Remaining <= 0 {
			log.Error().Err(err).Msg("rate limit has been already exhausted")
			break
		}

		log.Info().
			Str("repository", repo.NameWithOwner).
			Str("githubId", repo.GithubId).
			Int("id", repo.Id).
			Msg("repairing history for repo")

		var totalDates []time.Time
		var totalStars int
		cursor := ""

		for {
			dates, info, err := (*job.loader).LoadRepoStarHistoryDates(repo.GithubId, cursor)
			if err != nil {
				log.Fatal().
					Err(err).
					Int("id", repo.Id).
					Str("repository", repo.NameWithOwner).
					Str("cursor", cursor).
					Msg("aborting loading star history")
			}

			cursor = info.NextCursor
			totalStars = info.TotalStars
			totalDates = append(totalDates, dates...)

			log.Info().
				Int("remainingLimit", info.RateLimitRemaining).
				Str("repository", repo.NameWithOwner).
				Msg("remaining rate limit")

			if dates[len(dates)-1].Before(untilDate) {
				break
			}

			if !info.HasNextPage {
				break
			}
		}

		starCounts := make(map[time.Time]int)
		cumulativeCounts := make(map[time.Time]int)

		countStars(&starCounts, totalDates)

		var keys []time.Time
		for date := range starCounts {
			keys = append(keys, date)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].After(keys[j])
		})

		cumulativeSum := totalStars
		for _, key := range keys {
			cumulativeSum -= starCounts[key]
			cumulativeCounts[key] = cumulativeSum
		}

		var inputs []repository.StarHistoryInput
		for key, value := range cumulativeCounts {
			inputs = append(inputs, repository.StarHistoryInput{
				Id:        repo.Id,
				CreatedAt: key,
				StarCount: value,
			})
		}

		err = job.historyRepository.BatchUpsertStarHistory(inputs)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to upsert star history %s", repo.NameWithOwner)
		}

		log.Info().Msgf("finished upserting star history for repo %s", repo.NameWithOwner)
	}

	log.Info().Msg("done repairing history")
}
