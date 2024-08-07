package github

import (
	"context"
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

const apiUrl = "https://api.github.com"

type GithubClient struct {
	rest    http.Client
	graphql graphql.Client
	ctx     context.Context
}

type authedTransport struct {
	wrapped      http.RoundTripper
	apiKey       string
	acceptHeader string
}

func NewClient(ctx context.Context, apiKey string) *GithubClient {
	httpClient := http.Client{
		Transport: &authedTransport{
			apiKey:       apiKey,
			acceptHeader: "application/vnd.github.star+json", // required for star history
			wrapped:      http.DefaultTransport,
		},
	}

	gqlClient := graphql.NewClient(apiUrl+"/graphql", &httpClient)

	return &GithubClient{
		ctx:     ctx,
		rest:    httpClient,
		graphql: gqlClient,
	}
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "bearer "+t.apiKey)
	req.Header.Set("Accept", t.acceptHeader)
	req.Header.Set("X-Github-Next-Global-ID", "1")
	return t.wrapped.RoundTrip(req)
}
