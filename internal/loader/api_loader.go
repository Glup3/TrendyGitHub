package loader

import (
	"context"
	"fmt"
	"sync"

	"github.com/glup3/TrendyGitHub/generated"
)

const (
	minStarCount = 50
	perPage      = 5
)

type APILoader struct {
	ctx    context.Context
	apiKey string
}

func NewAPILoader(ctx context.Context, apiKey string) *APILoader {
	return &APILoader{ctx: ctx, apiKey: apiKey}
}

func (l *APILoader) LoadRepos(maxStarCount int, cursor string) ([]GitHubRepo, *PageInfo, error) {
	client := GetApiClient(l.apiKey)

	resp, err := generated.GetPublicRepos(l.ctx, client, fmt.Sprintf("is:public stars:%d..%d", minStarCount, maxStarCount), perPage, cursor)
	if err != nil {
		return nil, nil, err
	}

	repos := make([]GitHubRepo, len(resp.Search.Edges))
	for i, edge := range resp.Search.Edges {
		repo, _ := edge.Node.(*generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepository)
		repos[i] = GitHubRepo{
			Id:              repo.Id,
			Name:            repo.Name,
			NameWithOwner:   repo.NameWithOwner,
			StarCount:       repo.StargazerCount,
			ForkCount:       repo.ForkCount,
			PrimaryLanguage: repo.PrimaryLanguage.Name,
			Languages:       mapLanguages(repo.Languages.Edges),
		}
	}

	pageInfo := &PageInfo{
		NextMaxStarCount: repos[len(repos)-1].StarCount,
		UnitCosts:        resp.RateLimit.Cost,
	}

	return repos, pageInfo, nil
}

func mapLanguages(edges []generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepositoryLanguagesLanguageConnectionEdgesLanguageEdge) []string {
	languages := []string{}

	for _, edge := range edges {
		if len(edge.Node.Name) > 0 {
			languages = append(languages, edge.Node.Name)
		}
	}

	return languages
}

func (l *APILoader) LoadMultipleRepos(maxStarCount int, cursors []string) ([]GitHubRepo, *PageInfo, error) {
	var wg sync.WaitGroup
	repoChan := make(chan []GitHubRepo, len(cursors))
	pageInfoChan := make(chan *PageInfo, len(cursors))
	errChan := make(chan error, len(cursors))

	loadReposWorker := func(cursor string) {
		defer wg.Done()
		repos, pageInfo, err := l.LoadRepos(maxStarCount, cursor)
		if err != nil {
			errChan <- err
			return
		}
		repoChan <- repos
		pageInfoChan <- pageInfo
	}

	for _, cursor := range cursors {
		wg.Add(1)
		go loadReposWorker(cursor)
	}

	wg.Wait()
	close(repoChan)
	close(pageInfoChan)
	close(errChan)

	var allRepos []GitHubRepo
	var allErrors []error
	smallestNextMaxStarCount := maxStarCount
	totalUnitCosts := 0

	for repos := range repoChan {
		allRepos = append(allRepos, repos...)
	}

	for pageInfo := range pageInfoChan {
		if pageInfo.NextMaxStarCount < smallestNextMaxStarCount {
			smallestNextMaxStarCount = pageInfo.NextMaxStarCount
		}

		totalUnitCosts += pageInfo.UnitCosts
	}

	for err := range errChan {
		allErrors = append(allErrors, err)
	}

	pageInfo := &PageInfo{
		NextMaxStarCount: smallestNextMaxStarCount,
		UnitCosts:        totalUnitCosts,
	}

	if len(allErrors) > 0 {
		return allRepos, pageInfo, fmt.Errorf("some fetches failed: %v", allErrors)
	}

	return allRepos, pageInfo, nil
}
