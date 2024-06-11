package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Khan/genqlient/graphql"
	"github.com/glup3/TrendyGitHub/generated"
	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
)

type GitHubRepository struct {
	id            string
	description   string
	name          string
	nameWithOwner string
	url           string
	languages     []string
	starsCount    int
	forksCount    int
}

type authedTransport struct {
	wrapped http.RoundTripper
	key     string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "bearer "+t.key)
	return t.wrapped.RoundTrip(req)
}

func main() {
	ctx := context.Background()

	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	db, err := database.NewDatabase(ctx, config.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	httpClient := http.Client{
		Transport: &authedTransport{
			key:     config.GitHubToken,
			wrapped: http.DefaultTransport,
		},
	}
	graphqlClient := graphql.NewClient("https://api.github.com/graphql", &httpClient)

	settings, err := database.LoadSettings(db, ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}

	resp, err := generated.GetPublicRepos(context.Background(), graphqlClient, settings.CursorValue)
	if err != nil {
		return
	}

	repos := make([]database.RepoInput, len(resp.Search.Edges))
	for i, edge := range resp.Search.Edges {
		repo, _ := edge.Node.(*generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepository)
		repos[i] = database.RepoInput{
			GithubId:      repo.Id,
			Name:          repo.Name,
			Url:           repo.Url,
			NameWithOwner: repo.NameWithOwner,
			StarCount:     repo.StargazerCount,
			ForkCount:     repo.ForkCount,
			Languages:     mapLanguages(repo.Languages.Edges),
		}
	}

	ids, err := database.UpsertRepositories(db, ctx, repos)
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Println(ids)

	nextCursor := resp.Search.PageInfo.EndCursor
	if !resp.Search.PageInfo.HasNextPage {
		nextCursor = ""
	}

	err = database.UpdateCursor(db, ctx, settings.ID, nextCursor)
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("Done")
}

func mapLanguages(edges []generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepositoryLanguagesLanguageConnectionEdgesLanguageEdge) []string {
	languages := make([]string, 5)

	for i, edge := range edges {
		languages[i] = edge.Node.Name
	}

	return languages
}
