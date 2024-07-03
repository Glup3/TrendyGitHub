package jobs

import (
	"context"
	"time"

	database "github.com/glup3/TrendyGitHub/internal/db"
	lo "github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/glup3/TrendyGitHub/internal/repository"
	"github.com/rs/zerolog/log"
)

type HistoryJob struct {
	loader            *lo.Loader
	historyRepository *repository.HistoryRepository
}

func NewHistoryJob(ctx context.Context, db *database.Database, dataLoader *lo.Loader) *HistoryJob {
	return &HistoryJob{
		loader:            dataLoader,
		historyRepository: repository.NewHistoryRepository(db, ctx),
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
