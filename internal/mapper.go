package config

import (
	"github.com/glup3/TrendyGitHub/internal/loader"
	"github.com/glup3/TrendyGitHub/internal/repository"
)

func MapGitHubRepoToInput(repo loader.GitHubRepo) repository.RepoInput {
	return repository.RepoInput{
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

func MapGitHubReposToInputs(repos []loader.GitHubRepo) []repository.RepoInput {
	inputs := make([]repository.RepoInput, len(repos))
	for i, repo := range repos {
		inputs[i] = MapGitHubRepoToInput(repo)
	}
	return inputs
}

func MapToLanguageInput(languages []loader.Language) []repository.LanguageInput {
	inputs := make([]repository.LanguageInput, len(languages))
	for i, lang := range languages {
		inputs[i] = repository.LanguageInput{
			Id:       lang.Name,
			Hexcolor: lang.Color,
		}
	}
	return inputs
}
