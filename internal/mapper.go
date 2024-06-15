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
		Languages:       repo.Languages,
		StarCount:       repo.StarCount,
		ForkCount:       repo.ForkCount,
		PrimaryLanguage: repo.PrimaryLanguage,
	}
}

func MapGitHubReposToInputs(repos []loader.GitHubRepo) []database.RepoInput {
	inputs := make([]database.RepoInput, len(repos))
	for i, repo := range repos {
		inputs[i] = MapGitHubRepoToInput(repo)
	}
	return inputs
}
