package db

import (
	"context"
	"fmt"
	"log"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type repoId = int32

type Settings struct {
	ID                     int
	CurrentMaxStarCount    int
	MinStarCount           int
	TimeoutSecondsPrevent  int
	TimeoutSecondsExceeded int
	TimeoutMaxUnits        int
	IsEnabled              bool
}

type RepoInput struct {
	GithubId        string
	Name            string
	NameWithOwner   string
	PrimaryLanguage string
	Description     string
	Languages       []string
	StarCount       int
	ForkCount       int
}

type StarHistoryInput struct {
	CreatedAt time.Time
	StarCount int
	Id        repoId
}

type MissingRepo struct {
	GithubId      string
	NameWithOwner string
	StarCount     int
	Id            repoId
}

const batchSize = 10_000

func LoadSettings(db *Database, ctx context.Context) (Settings, error) {
	var settings Settings

	selectBuilder := sq.StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select(
			"id",
			"current_max_star_count",
			"min_star_count",
			"timeout_seconds_prevent",
			"timeout_seconds_exceeded",
			"timeout_max_units",
			"enabled",
		).
		From("settings").
		Limit(1)

	sql, args, err := selectBuilder.ToSql()
	if err != nil {
		return settings, fmt.Errorf("error building SQL: %v", err)
	}

	err = db.pool.QueryRow(ctx, sql, args...).
		Scan(
			&settings.ID,
			&settings.CurrentMaxStarCount,
			&settings.MinStarCount,
			&settings.TimeoutSecondsPrevent,
			&settings.TimeoutSecondsExceeded,
			&settings.TimeoutMaxUnits,
			&settings.IsEnabled,
		)
	if err != nil {
		return settings, fmt.Errorf("error loading settings: %v", err)
	}

	return settings, nil
}

func UpsertRepositories(db *Database, ctx context.Context, repos []RepoInput) ([]repoId, error) {
	upsertBuilder := sq.Insert("repositories").
		Columns(
			"github_id",
			"name",
			"name_with_owner",
			"star_count",
			"fork_count",
			"languages",
			"primary_language",
			"description",
		)

	for _, repo := range repos {
		upsertBuilder = upsertBuilder.Values(
			repo.GithubId,
			repo.Name,
			repo.NameWithOwner,
			repo.StarCount,
			repo.ForkCount,
			repo.Languages,
			repo.PrimaryLanguage,
			repo.Description,
		)
	}

	sql, args, err := upsertBuilder.
		Suffix(`
			ON CONFLICT (github_id)
			DO UPDATE SET
				star_count = EXCLUDED.star_count,
				fork_count = EXCLUDED.fork_count,
        primary_language = EXCLUDED.primary_language,
				languages = EXCLUDED.languages,
        description = EXCLUDED.description
		`).
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

func UpdateCurrentMaxStarCount(db *Database, ctx context.Context, settingsID int, newMaxStarCount int) error {
	updateBuilder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("settings").
		Set("current_max_star_count", newMaxStarCount).
		Where(sq.Eq{"id": settingsID})

	sql, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("error building SQL: %v", err)
	}

	commandTag, err := db.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("error updating current max star count in SQL: %v", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("no rows were updated")
	}

	return nil
}

func createStarHistorySnapshot(tx pgx.Tx, ctx context.Context) (int64, error) {
	sql, args, err := sq.Insert("stars_history").
		Columns("repository_id", "star_count", "created_at").
		Select(sq.Select("id", "star_count", "CURRENT_DATE").From("repositories")).
		Suffix(`
      ON CONFLICT (repository_id, created_at)
      DO UPDATE SET
      star_count = EXCLUDED.star_count
    `).
		ToSql()
	if err != nil {
		return 0, err
	}

	commandTag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}

	return commandTag.RowsAffected(), nil
}

func resetMaxStarCount(tx pgx.Tx, ctx context.Context, settingsId int) error {
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("settings").
		Set("current_max_star_count", 1_000_000).
		Where(sq.Eq{"id": settingsId}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func CreateSnapshotAndReset(db *Database, ctx context.Context, settingsId int) (int64, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %v", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p) // Re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback(ctx)
			fmt.Println("transaction failed -> rollback:", err)
		} else {
			err = tx.Commit(ctx)
			if err != nil {
				fmt.Println("commit failed:", err)
			}
		}
	}()

	_, err = tx.Exec(ctx, "SET LOCAL work_mem TO '128MB'")
	if err != nil {
		return 0, fmt.Errorf("failed to set work_mem: %v", err)
	}

	rowsUpdated, err := createStarHistorySnapshot(tx, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create star history snapshot: %v", err)
	}

	err = resetMaxStarCount(tx, ctx, settingsId)
	if err != nil {
		return 0, fmt.Errorf("failed to reset max star count: %v", err)
	}

	return rowsUpdated, nil
}

func GetGitHubId(db *Database, ctx context.Context, id repoId) (string, error) {
	var githubId string

	sql, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("github_id").
		From("repositories").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		log.Fatal(err)
	}

	err = db.pool.QueryRow(ctx, sql, args...).Scan(&githubId)
	if err != nil {
		return githubId, err
	}

	return githubId, nil
}

func BatchUpsertStarHistory(db *Database, ctx context.Context, inputs []StarHistoryInput) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	for start := 0; start < len(inputs); start += batchSize {
		end := start + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}

		builder := builder.Insert("stars_history").Columns("repository_id", "star_count", "created_at")

		for _, input := range inputs[start:end] {
			builder = builder.Values(input.Id, input.StarCount, input.CreatedAt)
		}

		sql, args, err := builder.
			Suffix(`
        ON CONFLICT (repository_id, created_at)
        DO UPDATE SET
        star_count = EXCLUDED.star_count
      `).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build SQL: %w", err)
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return fmt.Errorf("failed to execute upsert: %w", err)
		}
	}

	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("repositories").
		Set("history_missing", false).
		Where(sq.Eq{"id": inputs[0].Id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func GetNextMissingHistoryIds(db *Database, ctx context.Context) ([]repoId, error) {
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id").
		From("repositories").
		Where(sq.Eq{"history_missing": true}).
		OrderBy("star_count asc").
		Limit(500).
		ToSql()
	if err != nil {
		return nil, err
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

func GetNextMissingHistoryRepo(db *Database, ctx context.Context, maxStarCount int) (MissingRepo, error) {
	var repo MissingRepo

	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "github_id", "star_count", "name_with_owner").
		From("repositories").
		Where(sq.Eq{"history_missing": true}).
		Where(sq.LtOrEq{"star_count": maxStarCount}).
		OrderBy("star_count desc").
		Limit(1).
		ToSql()
	if err != nil {
		return repo, err
	}

	err = db.pool.QueryRow(ctx, sql, args...).Scan(&repo.Id, &repo.GithubId, &repo.StarCount, &repo.NameWithOwner)
	if err != nil {
		return repo, err
	}

	return repo, nil
}

func RefreshHistoryView(db *Database, ctx context.Context, viewName string) error {
	sqlStr := fmt.Sprintf("REFRESH MATERIALIZED VIEW %s", pgx.Identifier{viewName}.Sanitize())

	_, err := db.pool.Exec(ctx, sqlStr)
	if err != nil {
		return err
	}

	return nil
}

func GetTotalRepoCount(db *Database, ctx context.Context) (int, error) {
	var count int
	sql, args, err := sq.Select("count(*)").From("repositories").ToSql()

	err = db.pool.QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
