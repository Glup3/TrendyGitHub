package main

import (
	"context"
	"os"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/jobs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if len(os.Args) < 2 {
		log.Fatal().Msg("Usage: ./tgh [search|history]")
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

	mode := os.Args[1]
	switch mode {
	case "search":
		jobs.SearchRepositories(db, ctx, configs.GitHubToken)
	case "history":
		jobs.RunHistory(db, ctx, configs.GitHubToken)
	default:
		log.Fatal().Msgf("Invalid mode: %s. Use 'search' or 'history'.\n", mode)
	}
}
