package db

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type repoId = int32

type Settings struct {
	CursorValue string
	ID          int
}

type RepoInput struct {
	GithubId      string
	Name          string
	Url           string
	NameWithOwner string
	Languages     []string
	StarCount     int
	ForkCount     int
}

func LoadSettings(db *Database, ctx context.Context) (Settings, error) {
	var settings Settings

	selectBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "cursor_value").
		From("settings").
		Limit(1)

	sql, args, err := selectBuilder.ToSql()
	if err != nil {
		return settings, fmt.Errorf("error building SQL: %v", err)
	}

	err = db.pool.QueryRow(ctx, sql, args...).Scan(&settings.ID, &settings.CursorValue)
	if err != nil {
		return settings, fmt.Errorf("error loading settings: %v", err)
	}

	return settings, nil
}

func UpsertRepositories(db *Database, ctx context.Context, repos []RepoInput) ([]repoId, error) {
	upsertBuilder := sq.Insert("repositories").
		Columns("github_id", "name", "url", "name_with_owner", "star_count", "fork_count", "languages")

	for _, repo := range repos {
		upsertBuilder = upsertBuilder.Values(repo.GithubId, repo.Name, repo.Url, repo.NameWithOwner, repo.StarCount, repo.ForkCount, repo.Languages)
		// Suffix("ON CONFLICT (github_id) DO UPDATE SET star_count = EXCLUDED.star_count, fork_count = EXCLUDED.fork_count, languages = EXCLUDED.languages;")
	}

	sql, args, err := upsertBuilder.
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("error building SQL: %v", err)
	}

	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	ids, err := pgx.CollectRows(rows, pgx.RowTo[repoId])
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func UpdateCursor(db *Database, ctx context.Context, settingsID int, nextCursor string) error {
	updateBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("settings").
		Set("cursor_value", nextCursor).
		Where(sq.Eq{"id": settingsID})

	sql, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("error building SQL: %v", err)
	}

	commandTag, err := db.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("error executing update cursor SQL: %v", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no rows were updated")
	}

	return nil
}
