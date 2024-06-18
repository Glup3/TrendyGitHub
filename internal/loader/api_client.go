package loader

import (
	"net/http"
	"sync"

	"github.com/Khan/genqlient/graphql"
)

var (
	graphqlClientInstance graphql.Client
	restClientInstance    *http.Client

	clientOnce     sync.Once
	restClientOnce sync.Once
)

type authedTransport struct {
	wrapped      http.RoundTripper
	apiKey       string
	acceptHeader string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "bearer "+t.apiKey)
	req.Header.Set("Accept", t.acceptHeader)
	req.Header.Set("X-Github-Next-Global-ID", "1")
	return t.wrapped.RoundTrip(req)
}

func GetApiClient(apiKey string) graphql.Client {
	clientOnce.Do(func() {
		httpClient := http.Client{
			Transport: &authedTransport{
				apiKey:       apiKey,
				acceptHeader: "application/json", // Default for GraphQL
				wrapped:      http.DefaultTransport,
			},
		}

		graphqlClientInstance = graphql.NewClient("https://api.github.com/graphql", &httpClient)
	})

	return graphqlClientInstance
}

func GetRestApiClient(apiKey string) *http.Client {
	restClientOnce.Do(func() {
		restClientInstance = &http.Client{
			Transport: &authedTransport{
				apiKey:       apiKey,
				acceptHeader: "application/vnd.github.star+json", // Required for REST
				wrapped:      http.DefaultTransport,
			},
		}
	})

	return restClientInstance
}
