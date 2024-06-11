package loader

type GitHubRepo struct {
	Id            string
	Description   string
	Name          string
	NameWithOwner string
	Url           string
	Languages     []string
	StarCount     int
	ForkCount     int
}

type PageInfo struct {
	NextCursor string
}

type DataLoader interface {
	LoadRepos(cursor string, maxStarCount int) ([]GitHubRepo, *PageInfo, error)
}
