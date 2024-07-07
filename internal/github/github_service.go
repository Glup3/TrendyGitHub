package github

import (
	"encoding/json"
	"net/http"
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
