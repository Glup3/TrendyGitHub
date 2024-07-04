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
	loader             *lo.Loader
	repoRepository     *repository.RepoRepository
	settingsRepository *repository.SettingsRepository
}

func NewRepoJob(ctx context.Context, db *database.Database, dataLoader *lo.Loader) *RepoJob {
	return &RepoJob{
		loader:             dataLoader,
		repoRepository:     repository.NewRepoRepository(ctx, db),
		settingsRepository: repository.NewSettingsRepository(ctx, db),
	}
}

func (job *RepoJob) Search() {
	unitCount := 0

	for {
		settings, err := job.settingsRepository.Load()
		if err != nil {
			log.Fatal().Err(err).Msgf("failed loading settings")
		}

		if !settings.IsEnabled {
			log.Info().Msg("repository crawling is disabled")
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
		repos, pageInfo, err := (*job.loader).LoadMultipleRepos(settings.CurrentMaxStarCount, pagination_100_based_cursors[:])
		if err != nil {
			if strings.Contains(err.Error(), "secondary") {
				rateLimited = true
			} else {
				log.Error().Err(err).Msg("something went wrong during loading")
			}
		}

		unitCount += pageInfo.UnitCosts

		inputs := config.MapGitHubReposToInputs(repos)
		err = job.repoRepository.UpsertMany(inputs)
		if err != nil {
			log.Error().Err(err).Msg("upserting failed - aborting")
		}

		err = job.repoRepository.UpsertLanguages(mapUniqueLanguages(repos))
		if err != nil {
			log.Warn().Err(err).Msg("ignore upsert language errors")
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

		err = job.settingsRepository.UpdateStarCountCursor(pageInfo.NextMaxStarCount, settings.ID)
		if err != nil {
			log.Fatal().Err(err).Msg("updating max star count failed")
		}
	}

	log.Info().Msg("done fetching repositories")
}

func (job *RepoJob) ResetStarCountCursor(settingsID int) {}

func mapUniqueLanguages(repos []lo.GitHubRepo) []repository.LanguageInput {
	var languageInputs []repository.LanguageInput
	languageMap := make(map[string]repository.LanguageInput)

	for _, repo := range repos {
		for _, lang := range config.MapToLanguageInput(repo.Languages) {
			if _, exists := languageMap[lang.Id]; !exists {
				languageInputs = append(languageInputs, lang)
				languageMap[lang.Id] = lang
			}
		}
	}

	return languageInputs
}
