package jobs

import (
	"context"
	"sort"
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
			Int32("id", repo.Id).
			Msg("repairing history for repo")

		var totalDates []time.Time
		var totalStars int
		cursor := ""

		for {
			dates, info, err := dataLoader.LoadRepoStarHistoryDates(repo.GithubId, cursor)
			if err != nil {
				log.Fatal().
					Err(err).
					Int32("id", repo.Id).
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

		var inputs []database.StarHistoryInput
		for key, value := range cumulativeCounts {
			inputs = append(inputs, database.StarHistoryInput{
				Id:        repo.Id,
				CreatedAt: key,
				StarCount: value,
			})
		}

		err = database.BatchUpsertStarHistory(db, ctx, inputs)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to upsert star history %s", repo.NameWithOwner)
		}

		log.Info().Msgf("finished upserting star history for repo %s", repo.NameWithOwner)
	}

	log.Info().Msg("done repairing history")
}
