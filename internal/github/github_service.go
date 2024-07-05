package github

import (
	"encoding/json"
	"net/http"
)

type RateLimit struct {
	remainingRest    int
	remainingGraphql int

	resetRest    int
	resetGraphql int
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

func (client githubClient) GetRateLimit() (RateLimit, error) {
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
		remainingRest:    rl.Rate.Remaining,
		remainingGraphql: rl.Resources.Graphql.Remaining,
		resetRest:        rl.Rate.Reset,
		resetGraphql:     rl.Resources.Graphql.Reset,
	}, nil
}
