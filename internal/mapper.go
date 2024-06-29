package config

import (
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/loader"
)

func MapGitHubRepoToInput(repo loader.GitHubRepo) database.RepoInput {
	return database.RepoInput{
		GithubId:        repo.Id,
		Name:            repo.Name,
		NameWithOwner:   repo.NameWithOwner,
		Languages:       MapLanguagesToStrings(repo.Languages),
		StarCount:       repo.StarCount,
		ForkCount:       repo.ForkCount,
		Description:     repo.Description,
		PrimaryLanguage: repo.PrimaryLanguage,
	}
}

func MapLanguagesToStrings(languages []loader.Language) []string {
	s := make([]string, len(languages))
	for i, lang := range languages {
		s[i] = lang.Name
	}
	return s
}

func MapGitHubReposToInputs(repos []loader.GitHubRepo) []database.RepoInput {
	inputs := make([]database.RepoInput, len(repos))
	for i, repo := range repos {
		inputs[i] = MapGitHubRepoToInput(repo)
	}
	return inputs
}

func MapToLanguageInput(languages []loader.Language) []database.LanguageInput {
	inputs := make([]database.LanguageInput, len(languages))
	for i, lang := range languages {
		inputs[i] = database.LanguageInput{
			Id:       lang.Name,
			Hexcolor: lang.Color,
		}
	}
	return inputs
}
