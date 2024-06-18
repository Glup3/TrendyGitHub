package jobs

import (
	"context"

	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/rs/zerolog/log"
)

func RefreshViews(db *database.Database, ctx context.Context) {
	log.Info().Msg("refreshing views...")

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
		log.Error().Errs("errors", errors).Msg("encountered issues refreshing views")
	}

	log.Info().Msg("done refreshing views")
}
