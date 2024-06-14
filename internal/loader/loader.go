package loader

type GitHubRepo struct {
	Id              string
	Description     string
	Name            string
	NameWithOwner   string
	Url             string
	PrimaryLanguage string
	Languages       []string
	StarCount       int
	ForkCount       int
}

type PageInfo struct {
	NextMaxStarCount int
	UnitCosts        int
}

type DataLoader interface {
	LoadRepos(maxStarCount int, cursor string) ([]GitHubRepo, *PageInfo, error)
	LoadMultipleRepos(maxStarCount int, cursors []string) ([]GitHubRepo, *PageInfo, error)
}
