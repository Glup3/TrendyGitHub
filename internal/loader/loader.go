package loader

import "time"

type Language struct {
	Name  string
	Color string
}

type GitHubRepo struct {
	Id              string
	Description     string
	Name            string
	NameWithOwner   string
	PrimaryLanguage string
	Languages       []Language
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

type StarHistoryHeader struct {
	NextPage int
	PrevPage int
	LastPage int
}

type DataLoader interface {
	LoadRepos(maxStarCount int, cursor string) ([]GitHubRepo, *PageInfo, error)
	LoadMultipleRepos(maxStarCount int, cursors []string) ([]GitHubRepo, *PageInfo, error)
	LoadRepoStarHistoryDates(githubId string, cursor string) ([]time.Time, *StarPageInfo, error)
	LoadRepoStarHistoryPage(repoNameWithOwner string, page int) ([]time.Time, *StarHistoryHeader, error)
	GetRateLimit() (*RateLimit, error)
	GetRateLimitRest() (*RateLimitRest, error)
}
