package main

import (
	"context"
	"os"
	"time"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/jobs"
	lo "github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if len(os.Args) < 2 {
		log.Fatal().Msg("Usage: ./tgh [search|history|history-40k]")
	}

	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("loading configuration failed")
	}

	db, err := database.NewDatabase(ctx, configs.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to connect to database")
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to ping database")
	}

	var loader lo.Loader
	loader = lo.NewAPILoader(ctx, configs.GitHubToken)
	searchJob := jobs.NewRepoJob(ctx, db, &loader)
	historyJob := jobs.NewHistoryJob(ctx, db, &loader)

	mode := os.Args[1]
	switch mode {
	case "search":
		searchJob.Search()

	case "history-40k":
		historyJob.FetchHistoryUnder40kStars()

	case "history":
		historyJob.FetchHistory()

	case "repair":
		date, err := time.Parse(time.DateOnly, "2024-06-05")
		if err != nil {
			log.Fatal().Err(err).Msg("formatting date failed")
		}
		historyJob.RepairHistory(date)

	case "refresh":
		historyJob.CreateSnapshot()
		searchJob.ResetStarCountCursor(1)
		historyJob.RefreshViews()

	default:
		log.Fatal().Msgf("Invalid mode: %s. Use 'search' or 'history' or 'history-40k'", mode)
	}
}
