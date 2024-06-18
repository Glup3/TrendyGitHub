package jobs

import (
	"context"
	"log"

	database "github.com/glup3/TrendyGitHub/internal/db"
)

func RefreshViews(db *database.Database, ctx context.Context) {
	log.Print("refreshing views...")

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
		log.Println(errors)
	}
}
