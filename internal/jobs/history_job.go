package jobs

import (
	"context"
	"fmt"
	"math"
	"sort"
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

func (job *HistoryJob) Repair40k() {
	repos, err := job.historyRepository.GetBrokenRepos(40_000)
	if err != nil {
		log.Fatal().Err(err).Msg("failed fetching repos 40k")
	}

	for _, repo := range repos {
		err := job.repair40k(repo)
		if err != nil {
			log.Fatal().
				Err(err).
				Int("id", repo.Id).
				Str("repository", repo.NameWithOwner).
				Msgf("repairing star history 40k failed %s", repo.NameWithOwner)
		}
	}

	log.Info().Msg("done repairing history 40k")
}

func (job *HistoryJob) Repair() {
	repos, err := job.historyRepository.GetBrokenRepos(1_000_000)
	if err != nil {
		log.Fatal().Err(err).Msg("failed fetching repos")
	}

	for _, repo := range repos {
		err := job.repair(repo)
		if err != nil {
			log.Fatal().
				Err(err).
				Int("id", repo.Id).
				Str("repository", repo.NameWithOwner).
				Msgf("repairing star history failed %s", repo.NameWithOwner)
		}
	}

	log.Info().Msg("done repairing history")
}

func (job *HistoryJob) repair40k(repo repository.BrokenRepo) error {
	rl, err := job.api.GetRateLimit()
	if err != nil {
		return err
	}
	if rl.RemainingRest <= 0 {
		return fmt.Errorf("next REST rate limit reset at %d", rl.ResetRest)
	}

	log.Info().
		Int("id", repo.Id).
		Str("repository", repo.NameWithOwner).
		Int("remaining", rl.RemainingRest).
		Msgf("repairing history 40k for repo %s", repo.NameWithOwner)

	var totalTimes []time.Time
	lastPage := int(math.Ceil(float64(repo.StarCount) / 100.0))

Pages:
	for page := lastPage; page >= 1; page-- {
		times, err := job.api.GetStarHistory(repo.NameWithOwner, page)
		if err != nil {
			return err
		}

		sort.Slice(times, func(i, j int) bool {
			return times[i].Before(times[j])
		})

		for _, time := range times {
			if time.Before(repo.UntilDate) {
				break Pages
			}

			totalTimes = append(totalTimes, time)
		}
	}

	if len(totalTimes) > 0 {
		err := job.updateAccumulatedStars(repo, totalTimes)
		if err != nil {
			return err
		}
	}

	err = job.historyRepository.RemoveBrokenRepo(repo.Id)
	if err != nil {
		return err
	}

	log.Info().
		Int("id", repo.Id).
		Str("repository", repo.NameWithOwner).
		Msgf("repaired star history 40k for %s", repo.NameWithOwner)

	return nil
}

func (job *HistoryJob) repair(repo repository.BrokenRepo) error {
	rl, err := job.api.GetRateLimit()
	if err != nil {
		return err
	}
	if rl.RemainingGraphql <= 0 {
		return fmt.Errorf("next Graphql rate limit reset at %d", rl.ResetGraphql)
	}

	log.Info().
		Int("id", repo.Id).
		Str("repository", repo.NameWithOwner).
		Int("remaining", rl.RemainingGraphql).
		Msgf("repairing history for repo %s", repo.NameWithOwner)

	var totalTimes []time.Time
	cursor := ""

Cursors:
	for cursor != "END" {
		times, nextCursor, err := job.api.GetStarHistoryV2(repo.GithubId, cursor)
		if err != nil {
			return err
		}

		cursor = nextCursor

		for _, time := range times {
			if time.Before(repo.UntilDate) {
				break Cursors
			}

			totalTimes = append(totalTimes, time)
		}
	}

	if len(totalTimes) > 0 {
		err := job.updateAccumulatedStars(repo, totalTimes)
		if err != nil {
			return err
		}
	}

	err = job.historyRepository.RemoveBrokenRepo(repo.Id)
	if err != nil {
		return err
	}

	log.Info().
		Int("id", repo.Id).
		Str("repository", repo.NameWithOwner).
		Msgf("repaired star history for %s", repo.NameWithOwner)

	return nil
}

func (job *HistoryJob) updateAccumulatedStars(repo repository.BrokenRepo, times []time.Time) error {
	if len(times) == 0 {
		return nil
	}

	baseStarCount, err := job.repoRepository.GetStarCount(repo.Id, repo.UntilDate.Add(-24*time.Hour))
	if err != nil {
		return err
	}

	starsByDate := aggregateStars(times)
	cumulativeCounts := accumulateStars(starsByDate, baseStarCount)
	var inputs []repository.StarHistoryInput
	for date, count := range cumulativeCounts {
		inputs = append(inputs, repository.StarHistoryInput{
			Id:        repo.Id,
			StarCount: count,
			Date:      date,
		})
	}

	err = job.historyRepository.BatchUpsert(inputs)
	if err != nil {
		return err
	}

	return nil
}
