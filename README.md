# TrendyGitHub

Tracks Trendy Public GitHub Repositories

## DB Migration

```sh
# load environment variables for migrate cli tool

source .env
```

`migrate -database "${DATABASE_URL}?sslmode=disable" -path ./db/migrations up`

`migrate -database "${DATABASE_URL}?sslmode=disable" -path ./db/migrations down`

## Notes

Primary rate limit: 5000 points / hour

Secondary rate limit: 2000 points burst for 1 minute

fetching 1 point (100 repositories) takes ~3 seconds

fetching 100 points (10_000 repositories) == ~300 seconds

since we fetch via limiting the max star count, the first entry is always the
last entry from the previous run

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
