package loader

import "time"

type GitHubRepo struct {
	Id              string
	Description     string
	Name            string
	NameWithOwner   string
	PrimaryLanguage string
	Languages       []string
	StarCount       int
	ForkCount       int
}

type PageInfo struct {
	NextMaxStarCount int
	UnitCosts        int
}

type StarPageInfo struct {
	NextCursor  string
	TotalStars  int
	HasNextPage bool
}

type DataLoader interface {
	LoadRepos(maxStarCount int, cursor string) ([]GitHubRepo, *PageInfo, error)
	LoadMultipleRepos(maxStarCount int, cursors []string) ([]GitHubRepo, *PageInfo, error)
	LoadRepoStarHistoryDates(cursor string) ([]time.Time, *StarPageInfo, error)
}
