package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/glup3/TrendyGitHub/generated"
)

const (
	minStarCount = 50
	perPage      = 100 // INFO: cursors depend on page size
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
			Description:     repo.Description,
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

func (l *APILoader) LoadRepoStarHistoryDates(githubId string, cursor string) ([]time.Time, *StarPageInfo, error) {
	client := GetApiClient(l.apiKey)

	resp, err := generated.GetStarGazers(l.ctx, client, githubId, cursor)
	if err != nil {
		return nil, nil, err
	}

	repo, _ := resp.Node.(*generated.GetStarGazersNodeRepository)
	dateTimes := make([]time.Time, len(repo.Stargazers.Edges))

	for i, stargazer := range repo.Stargazers.Edges {
		dateTimes[i] = stargazer.StarredAt
	}

	pageInfo := &StarPageInfo{
		TotalStars:         repo.Stargazers.TotalCount,
		NextCursor:         repo.Stargazers.PageInfo.EndCursor,
		HasNextPage:        repo.Stargazers.PageInfo.HasNextPage,
		RateLimitRemaining: resp.RateLimit.Remaining,
		RateLimitResetAt:   resp.RateLimit.ResetAt,
	}

	return dateTimes, pageInfo, nil
}

type Stargazer struct {
	StarredAt time.Time `json:"starred_at"`
}

// page is 1-based
func (l *APILoader) LoadRepoStarHistoryPage(repoNameWithOwner string, page int) ([]time.Time, *StarHistoryHeader, error) {
	client := GetRestApiClient(l.apiKey)
	if client == nil {
		return nil, nil, fmt.Errorf("failed to get HTTP client")
	}

	var dateTimes []time.Time
	var pageInfo StarHistoryHeader

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/stargazers?page=%d&per_page=100", repoNameWithOwner, page), nil)
	if err != nil {
		return dateTimes, nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return dateTimes, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return dateTimes, nil, fmt.Errorf("failed to fetch stargazers: %s", resp.Status)
	}

	var rawStargazers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawStargazers); err != nil {
		return dateTimes, nil, err
	}

	for _, stargazer := range rawStargazers {
		if starredAtStr, ok := stargazer["starred_at"].(string); ok {
			if starredAt, err := time.Parse(time.RFC3339, starredAtStr); err == nil {
				dateTimes = append(dateTimes, starredAt)
			}
		}
	}

	if linkHeader := resp.Header.Get("Link"); linkHeader != "" {
		pageInfo = parseLinkHeader(linkHeader)
	}

	return dateTimes, &pageInfo, nil
}

func parseLinkHeader(linkHeader string) StarHistoryHeader {
	var pageInfo StarHistoryHeader

	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(link, ";")
		if len(parts) != 2 {
			continue
		}
		urlPart := strings.Trim(parts[0], "<> ")
		relPart := strings.TrimSpace(parts[1])

		u, err := url.Parse(urlPart)
		if err != nil {
			continue
		}

		page := extractPageFromURL(u)
		if strings.Contains(relPart, `rel="next"`) {
			pageInfo.NextPage = page
		} else if strings.Contains(relPart, `rel="prev"`) {
			pageInfo.PrevPage = page
		} else if strings.Contains(relPart, `rel="last"`) {
			pageInfo.LastPage = page
		}
	}

	return pageInfo
}

// extractPageFromURL extracts the page number from a URL query parameter
func extractPageFromURL(u *url.URL) int {
	q := u.Query()
	pageStr := q.Get("page")
	page, _ := strconv.Atoi(pageStr)
	return page
}

func (l *APILoader) GetRateLimit() (*RateLimit, error) {
	client := GetApiClient(l.apiKey)

	resp, err := generated.GetRateLimit(l.ctx, client)
	if err != nil {
		return nil, err
	}

	rateLimit := &RateLimit{
		Remaining: resp.RateLimit.Remaining,
		Used:      resp.RateLimit.Used,
		Limit:     resp.RateLimit.Limit,
		ResetAt:   resp.RateLimit.ResetAt,
	}

	return rateLimit, nil
}
