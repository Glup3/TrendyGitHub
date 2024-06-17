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
	RateLimitResetAt   time.Time
	NextCursor         string
	TotalStars         int
	RateLimitRemaining int
	HasNextPage        bool
}

type RateLimit struct {
	ResetAt   time.Time
	Limit     int
	Remaining int
	Used      int
}

type DataLoader interface {
	LoadRepos(maxStarCount int, cursor string) ([]GitHubRepo, *PageInfo, error)
	LoadMultipleRepos(maxStarCount int, cursors []string) ([]GitHubRepo, *PageInfo, error)
	LoadRepoStarHistoryDates(githubId string, cursor string) ([]time.Time, *StarPageInfo, error)
	GetRateLimit() (*RateLimit, error)
}
