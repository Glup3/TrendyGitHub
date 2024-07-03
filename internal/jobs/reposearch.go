package jobs

import (
	"context"
	"strings"
	"time"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
	lo "github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/glup3/TrendyGitHub/internal/repository"
	"github.com/rs/zerolog/log"
)

var pagination_100_based_cursors = [...]string{
	"",
	"Y3Vyc29yOjEwMA==",
	"Y3Vyc29yOjIwMA==",
	"Y3Vyc29yOjMwMA==",
	"Y3Vyc29yOjQwMA==",
	"Y3Vyc29yOjUwMA==",
	"Y3Vyc29yOjYwMA==",
	"Y3Vyc29yOjcwMA==",
	"Y3Vyc29yOjgwMA==",
	"Y3Vyc29yOjkwMA==",
}

type RepoJob struct {
	loader         *lo.Loader
	repoRepository *repository.RepoRepository
}

func NewRepoJob(ctx context.Context, db *database.Database, dataLoader *lo.Loader) *RepoJob {
	return &RepoJob{
		loader:         dataLoader,
		repoRepository: repository.NewRepoRepository(ctx, db),
	}
}

func (j *RepoJob) Search(db *database.Database, ctx context.Context) {
	unitCount := 0

	defer func() {
		log.Info().Msg("creating snapshot and resetting")

		rows, err := database.CreateSnapshotAndReset(db, ctx, 1)
		if err != nil {
			log.Fatal().Err(err).Msg("failed updating snapshot")
		}

		log.Info().Msgf("created snapshot for %d repositories", rows)

		RefreshViews(db, ctx)
	}()

	for {
		settings, err := database.LoadSettings(db, ctx)
		if err != nil {
			log.Fatal().Err(err).Msgf("error loading settings - aborting")
		}

		if !settings.IsEnabled {
			log.Info().Msg("repository crawling is disabled - aborting")
			break
		}

		if settings.CurrentMaxStarCount <= settings.MinStarCount {
			log.Info().Msg("reached the end - no more data loading")
			break
		}

		if unitCount >= settings.TimeoutMaxUnits {
			log.Info().Msgf("rate limit prevention - waiting %d seconds", settings.TimeoutSecondsPrevent)
			time.Sleep(time.Duration(settings.TimeoutSecondsPrevent) * time.Second)
			unitCount = 0
		}

		log.Info().Msgf("started fetching stars >= %d", settings.CurrentMaxStarCount)

		rateLimited := false
		repos, pageInfo, err := (*j.loader).LoadMultipleRepos(settings.CurrentMaxStarCount, pagination_100_based_cursors[:])
		if err != nil {
			if strings.Contains(err.Error(), "secondary") {
				rateLimited = true
			} else {
				log.Error().Err(err).Msg("something went wrong during loading")
			}
		}

		unitCount += pageInfo.UnitCosts

		inputs := config.MapGitHubReposToInputs(repos)
		err = j.repoRepository.UpsertMany(inputs)
		// _, err = database.UpsertRepositories(db, ctx, inputs)
		if err != nil {
			log.Error().Err(err).Msg("upserting failed - aborting")
		}

		var languageInputs []database.LanguageInput
		languageMap := make(map[string]database.LanguageInput)

		for _, repo := range repos {
			for _, lang := range config.MapToLanguageInput(repo.Languages) {
				if _, exists := languageMap[lang.Id]; !exists {
					languageInputs = append(languageInputs, lang)
					languageMap[lang.Id] = lang
				}
			}
		}
		if err = database.UpsertLanguages(db, ctx, languageInputs); err != nil {
			log.Warn().
				Err(err).
				Msg("ignore upsert language errors")
		}

		if rateLimited {
			log.Info().Msgf("got rate limited - waiting %d seconds", settings.TimeoutSecondsExceeded)
			time.Sleep(time.Duration(settings.TimeoutSecondsExceeded) * time.Second)
			unitCount = 0
			continue
		}

		if settings.CurrentMaxStarCount == pageInfo.NextMaxStarCount {
			log.Warn().Msgf("next max star count is equal current max star count - %d", pageInfo.NextMaxStarCount)
			break
		}

		err = database.UpdateCurrentMaxStarCount(db, ctx, settings.ID, pageInfo.NextMaxStarCount)
		if err != nil {
			log.Fatal().Err(err).Msg("updating max star count failed")
		}
	}

	log.Info().Msg("done fetching repositories")
}
