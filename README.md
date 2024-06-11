# TrendyGitHub

Tracks Trendy Public GitHub Repositories

## DB Migration

`migrate -database "${DATABASE_URL}?sslmode=disable" -path ./db/migrations up`

`migrate -database "${DATABASE_URL}?sslmode=disable" -path ./db/migrations down`

## Roadmap

MVP Alpha

- [ ] Fetch repo data from GitHub
- [ ] Automated fetching respecting rate limits
- [ ] Store data in database
- [ ] Aggregate star counts into trends
- [ ] Scheduled execution of fetching and aggregating

MVP Beta

- [ ] REST API for trends
- [ ] Rate limiting API
- [ ] Cache trends

V1

- [ ] Homepage

Maybe?

- [ ] CLI
