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

*NEW*
After increasing speed by 10x, I am getting secondary rate limited at about
66 requests in 1 minute against the same endpoint

Primary rate limit: 5000 points / hour

Secondary rate limit: 2000 points burst for 1 minute

fetching 1 point (100 repositories) takes ~3 seconds

fetching 100 points (10_000 repositories) == ~300 seconds

since we fetch via limiting the max star count, the first entry is always the
last entry from the previous run

12th June 2024

| iteration | star count |
|-----------|------------|
| 300 | 1694 |
| 400 | 1310|
| 500 | 1057 |
| 600 | 904 |
| 700 | 803 |
| 800 | 728 |

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
