package loader

import (
	"context"

	"github.com/glup3/TrendyGitHub/generated"
)

type APILoader struct {
	ctx    context.Context
	apiKey string
}

func NewAPILoader(ctx context.Context, apiKey string) *APILoader {
	return &APILoader{ctx: ctx, apiKey: apiKey}
}

func (l *APILoader) LoadRepos(cursor string) ([]GitHubRepo, *PageInfo, error) {
	client := GetApiClient(l.apiKey)

	resp, err := generated.GetPublicRepos(l.ctx, client, cursor)
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

	nextCursor := resp.Search.PageInfo.EndCursor
	if !resp.Search.PageInfo.HasNextPage {
		nextCursor = ""
	}

	pageInfo := &PageInfo{
		NextCursor: nextCursor,
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
