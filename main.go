package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Khan/genqlient/graphql"
	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/generated"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
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

type Settings struct {
	CursorValue string
	ID          int
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
	var err error
	defer func() {
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	err = godotenv.Load()
	if err != nil {
		err = fmt.Errorf("loading .env file failed: %v", err)
		return
	}

	key := os.Getenv("GITHUB_TOKEN")
	if key == "" {
		err = fmt.Errorf("must set GITHUB_TOKEN=<github token>")
		return
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		err = fmt.Errorf("DATABASE_URL not set")
		return
	}

	fmt.Println("Connecting to database...")

	pool, err := pgxpool.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	httpClient := http.Client{
		Transport: &authedTransport{
			key:     key,
			wrapped: http.DefaultTransport,
		},
	}
	graphqlClient := graphql.NewClient("https://api.github.com/graphql", &httpClient)

	// Build the select query using Squirrel
	selectBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "cursor_value").
		From("settings").
		Limit(1)

	// Execute the query
	sql, args, err := selectBuilder.ToSql()
	if err != nil {
		log.Fatalf("Error building SQL: %v", err)
	}

	fmt.Println("Loading settings...")

	var settings Settings
	err = pool.QueryRow(context.Background(), sql, args...).Scan(&settings.ID, &settings.CursorValue)
	if err != nil {
		log.Fatalf("Error querying settings: %v", err)
	}

	fmt.Println("Loading repos...")

	var repoResp *generated.GetPublicReposResponse

	repoResp, err = generated.GetPublicRepos(context.Background(), graphqlClient, settings.CursorValue)
	if err != nil {
		return
	}

	upsertBuilder := sq.Insert("repositories").Columns("github_id", "name", "url", "name_with_owner", "star_count", "fork_count", "languages")

	for _, edge := range repoResp.Search.Edges {
		repo, ok := edge.Node.(*generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepository)
		if !ok {
			// ignore error
			continue
		}

		upsertBuilder = upsertBuilder.Values(repo.Id, repo.Name, repo.Url, repo.NameWithOwner, repo.StargazerCount, repo.ForkCount, mapLanguages(repo.Languages.Edges))
		// Suffix("ON CONFLICT (github_id) DO UPDATE SET star_count = EXCLUDED.star_count, fork_count = EXCLUDED.fork_count, languages = EXCLUDED.languages;")
	}

	sql, args, err = upsertBuilder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		log.Fatalf("Error building SQL: %v", err)
	}

	fmt.Println("Inserting repos...")

	// Execute the UPSERT query
	_, err = pool.Exec(context.Background(), sql, args...)
	if err != nil {
		log.Fatalf("Error performing UPSERT: %v", err)
	}

	nextCursor := repoResp.Search.PageInfo.EndCursor
	if !repoResp.Search.PageInfo.HasNextPage {
		nextCursor = ""
	}

	updateBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("settings").
		Set("cursor_value", nextCursor).
		Where(sq.Eq{"id": settings.ID})

	// Execute the query
	sql, args, err = updateBuilder.ToSql()
	if err != nil {
		log.Fatalf("Error building SQL: %v", err)
	}

	fmt.Println("Updating cursor...")

	_, err = pool.Exec(context.Background(), sql, args...)
	if err != nil {
		log.Fatalf("Error updating cursor value: %v", err)
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
