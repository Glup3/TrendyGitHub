package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/glup3/TrendyGitHub/generated"
	"github.com/joho/godotenv"
)

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

	httpClient := http.Client{
		Transport: &authedTransport{
			key:     key,
			wrapped: http.DefaultTransport,
		},
	}
	graphqlClient := graphql.NewClient("https://api.github.com/graphql", &httpClient)

	var repoResp *generated.GetPublicReposResponse

	repoResp, err = generated.GetPublicRepos(context.Background(), graphqlClient, "")
	if err != nil {
		return
	}

	// loop over the response search edges and write the url into a txt file
	for _, v := range repoResp.Search.Edges {
		repo, ok := v.Node.(*generated.GetPublicReposSearchSearchResultItemConnectionEdgesSearchResultItemEdgeNodeRepository)
		if !ok {
			// ignore error
			continue
		}

		fmt.Println(repo.Name, " | ", repo.Url, " | ", repo.Id)
	}
}
