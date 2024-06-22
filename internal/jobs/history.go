package jobs

import (
	"context"
	"time"

	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/rs/zerolog/log"
)

func RepairHistory(db *database.Database, ctx context.Context, githubToken string, untilDate time.Time) {
	dataLoader := loader.NewAPILoader(ctx, githubToken)

	repos, err := database.GetAllPresentHistoryRepos(db, ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed fetching repos")
	}

	if len(repos) == 0 {
		log.Info().Msg("no repos to repair history for")
	}

	for _, repo := range repos {
		rateLimit, err := dataLoader.GetRateLimit()
		if err != nil {
			log.Fatal().Err(err).Msg("failed fetching GraphQL API rate limit")
		}

		if rateLimit.Remaining <= 0 {
			log.Fatal().Err(err).Msg("rate limit has been already exhausted")
		}

		log.Info().
			Str("repository", repo.NameWithOwner).
			Str("githubId", repo.GithubId).
			Int32("id", repo.Id).
			Msg("repairing history for repo")

		var totalDates []time.Time
		cursor := ""

		for {
			dates, info, err := dataLoader.LoadRepoStarHistoryDates(repo.GithubId, cursor)
			if err != nil {
				log.Fatal().
					Err(err).
					Str("repository", repo.NameWithOwner).
					Str("githubId", repo.GithubId).
					Int32("id", repo.Id).
					Str("cursor", cursor).
					Msg("aborting loading star history")
			}

			cursor = info.NextCursor
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

		AggregateAndInsertHistory(db, ctx, totalDates, database.MissingRepo{
			NameWithOwner: repo.NameWithOwner,
			Id:            repo.Id,
		})
	}

	RefreshViews(db, ctx)

	log.Info().Msg("done repairing history")
}
