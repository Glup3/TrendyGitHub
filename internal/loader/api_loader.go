package loader

import (
	"context"
	"fmt"

	"github.com/glup3/TrendyGitHub/generated"
)

const (
	minStarCount = 50
	perPage      = 100
)

type APILoader struct {
	ctx    context.Context
	apiKey string
}

func NewAPILoader(ctx context.Context, apiKey string) *APILoader {
	return &APILoader{ctx: ctx, apiKey: apiKey}
}

func (l *APILoader) LoadRepos(maxStarCount int) ([]GitHubRepo, *PageInfo, error) {
	client := GetApiClient(l.apiKey)

	resp, err := generated.GetPublicRepos(l.ctx, client, fmt.Sprintf("is:public stars:%d..%d", minStarCount, maxStarCount), perPage)
	if err != nil {
		return nil, nil, err
	}

	repos := make([]GitHubRepo, len(resp.Search.Edges))
	for i, edge := range resp.Search.Edges {
		repo, _ := edge.Node.(*generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepository)
		repos[i] = GitHubRepo{
			Id:            repo.Id,
			Name:          repo.Name,
			Url:           repo.Url,
			NameWithOwner: repo.NameWithOwner,
			StarCount:     repo.StargazerCount,
			ForkCount:     repo.ForkCount,
			Languages:     mapLanguages(repo.Languages.Edges),
		}
	}

	pageInfo := &PageInfo{
		NextMaxStarCount: repos[len(repos)-1].StarCount,
	}

	return repos, pageInfo, nil
}

func mapLanguages(edges []generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepositoryLanguagesLanguageConnectionEdgesLanguageEdge) []string {
	languages := make([]string, 5)

	for i, edge := range edges {
		languages[i] = edge.Node.Name
	}

	return languages
}
