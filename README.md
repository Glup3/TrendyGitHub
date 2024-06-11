# TrendyGitHub

Tracks Trendy Public GitHub Repositories

## DB Migration

`migrate -database "${DATABASE_URL}?sslmode=disable" -path ./db/migrations up`
`migrate -database "${DATABASE_URL}?sslmode=disable" -path ./db/migrations down`
