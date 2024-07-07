package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RateLimit struct {
	RemainingRest    int
	RemainingGraphql int

	ResetRest    int
	ResetGraphql int
}

type rateLimit struct {
	Resources struct {
		Graphql struct {
			Limit     int `json:"limit"`
			Used      int `json:"used"`
			Remaining int `json:"remaining"`
			Reset     int `json:"reset"`
		} `json:"graphql"`
	} `json:"resources"`
	Rate struct {
		Limit     int `json:"limit"`
		Used      int `json:"used"`
		Remaining int `json:"remaining"`
		Reset     int `json:"reset"`
	} `json:"rate"`
}

type stargazer struct {
	StarredAt time.Time `json:"starred_at"`
}

func (client GithubClient) GetRateLimit() (RateLimit, error) {
	req, err := http.NewRequest("GET", apiUrl+"/rate_limit", nil)
	if err != nil {
		return RateLimit{}, err
	}

	resp, err := client.rest.Do(req)
	if err != nil {
		return RateLimit{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RateLimit{}, err
	}

	var rl rateLimit
	err = json.NewDecoder(resp.Body).Decode(&rl)
	if err != nil {
		return RateLimit{}, err
	}

	return RateLimit{
		RemainingRest:    rl.Rate.Remaining,
		RemainingGraphql: rl.Resources.Graphql.Remaining,
		ResetRest:        rl.Rate.Reset,
		ResetGraphql:     rl.Resources.Graphql.Reset,
	}, nil
}

func (client GithubClient) GetStarHistory(repoFullName string, page int) ([]time.Time, error) {
	var times []time.Time

	url := fmt.Sprintf("%s/repos/%s/stargazers?page=%d&per_page=100", apiUrl, repoFullName, page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return times, err
	}

	resp, err := client.rest.Do(req)
	if err != nil {
		return times, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return times, err
	}

	var stargazers []stargazer
	err = json.NewDecoder(resp.Body).Decode(&stargazers)
	if err != nil {
		return times, err
	}

	for _, stargazer := range stargazers {
		times = append(times, stargazer.StarredAt)
	}

	return times, nil
}
