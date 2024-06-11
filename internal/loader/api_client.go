package loader

import (
	"net/http"
	"sync"

	"github.com/Khan/genqlient/graphql"
)

var (
	clientInstance graphql.Client

	clientOnce sync.Once
)

type authedTransport struct {
	wrapped http.RoundTripper
	apiKey  string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "bearer "+t.apiKey)
	return t.wrapped.RoundTrip(req)
}

func GetApiClient(apiKey string) graphql.Client {
	clientOnce.Do(func() {
		httpClient := http.Client{
			Transport: &authedTransport{
				apiKey:  apiKey,
				wrapped: http.DefaultTransport,
			},
		}

		clientInstance = graphql.NewClient("https://api.github.com/graphql", &httpClient)
	})

	return clientInstance
}
