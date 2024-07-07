package jobs

import (
	"context"
	"math"
	"time"

	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/github"
	lo "github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/glup3/TrendyGitHub/internal/repository"
	"github.com/rs/zerolog/log"
)

type HistoryJob struct {
	loader            *lo.Loader
	repoRepository    *repository.RepoRepository
	historyRepository *repository.HistoryRepository
	api               *github.GithubClient
}

func NewHistoryJob(ctx context.Context, db *database.Database, dataLoader *lo.Loader, githubClient *github.GithubClient) *HistoryJob {
	return &HistoryJob{
		loader:            dataLoader,
		historyRepository: repository.NewHistoryRepository(ctx, db),
		repoRepository:    repository.NewRepoRepository(ctx, db),
		api:               githubClient,
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
		log.Info().Msgf("refreshing %s took %s", view, elapsed)
	}

	log.Info().Msgf("refreshing views took %s", time.Since(start))
}

func (job *HistoryJob) Repair40k(untilDate time.Time) {
	repos, err := job.historyRepository.GetBrokenRepos(40_000)
	if err != nil {
		log.Fatal().Err(err).Msg("failed fetching repos")
	}

	for _, repo := range repos {
		rl, err := job.api.GetRateLimit()
		if err != nil {
			log.Error().Err(err).Str("job", "repair").Msg("failed fetching rate limits")
			break
		}
		if rl.RemainingRest <= 0 {
			log.Info().Str("job", "repair").Int("resetAt", rl.ResetRest).Msgf("next REST rate limit reset at %d", rl.ResetRest)
		}

		log.Info().
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msgf("repairing history for repo %s", repo.NameWithOwner)

		var totalTimes []time.Time
		lastPage := int(math.Ceil(float64(repo.StarCount / 100.0)))

	Pages:
		for page := lastPage; page >= 0; page-- {
			times, err := job.api.GetStarHistory(repo.NameWithOwner, page)
			if err != nil {
				log.Fatal().
					Err(err).
					Int("id", repo.Id).
					Str("repository", repo.NameWithOwner).
					Int("page", page).
					Msg("abort repairing star history")
			}

			for _, time := range times {
				if time.Before(untilDate) {
					break Pages
				}

				totalTimes = append(totalTimes, time)
			}
		}

		baseStarCount := 0
		starsByDate := aggregateStars(totalTimes)
		cumulativeCounts := accumulateStars(starsByDate, baseStarCount)

		var inputs []repository.StarHistoryInput
		for date, count := range cumulativeCounts {
			inputs = append(inputs, repository.StarHistoryInput{
				Id:        repo.Id,
				StarCount: count,
				CreatedAt: date,
			})
		}

		err = job.historyRepository.BatchUpsertStarHistory(inputs)
		if err != nil {
			log.Fatal().
				Err(err).
				Int("id", repo.Id).
				Str("repository", repo.NameWithOwner).
				Msgf("failed to repair star history for %s", repo.NameWithOwner)
		}

		log.Info().
			Int("id", repo.Id).
			Str("repository", repo.NameWithOwner).
			Msgf("repaired star history for %s", repo.NameWithOwner)
	}

	log.Info().Msg("done repairing history")
}
