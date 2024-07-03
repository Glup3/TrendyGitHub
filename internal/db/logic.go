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

type PresentRepo struct {
	GithubId      string
	NameWithOwner string
	Id            repoId
}

const batchSize = 10_000

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

	err = db.Pool.QueryRow(ctx, sql, args...).Scan(&githubId)
	if err != nil {
		return githubId, err
	}

	return githubId, nil
}

func BatchUpsertStarHistory(db *Database, ctx context.Context, inputs []StarHistoryInput) error {
	tx, err := db.Pool.Begin(ctx)
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
        star_count = EXCLUDED.star_count,
        created_at = EXCLUDED.created_at
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

func MarkRepoAsDone(db *Database, ctx context.Context, id repoId) error {
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("repositories").
		Set("history_missing", false).
		Where(sq.Eq{"id": id}).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	_, err = db.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
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

	rows, err := db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	ids, err := pgx.CollectRows(rows, pgx.RowTo[repoId])
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func GetNextMissingHistoryRepo(db *Database, ctx context.Context, maxStarCount int, ascendingOrder bool) (MissingRepo, error) {
	var repo MissingRepo

	orderDirection := "desc"
	if ascendingOrder {
		orderDirection = "asc"
	}

	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "github_id", "star_count", "name_with_owner").
		From("repositories").
		Where(sq.Eq{"history_missing": true}).
		Where(sq.LtOrEq{"star_count": maxStarCount}).
		OrderBy("star_count " + orderDirection).
		Limit(1).
		ToSql()
	if err != nil {
		return repo, err
	}

	err = db.Pool.QueryRow(ctx, sql, args...).Scan(&repo.Id, &repo.GithubId, &repo.StarCount, &repo.NameWithOwner)
	if err != nil {
		return repo, err
	}

	return repo, nil
}

func GetTotalRepoCount(db *Database, ctx context.Context) (int, error) {
	var count int
	sql, args, err := sq.Select("count(*)").From("repositories").ToSql()

	err = db.Pool.QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetAllPresentHistoryRepos(db *Database, ctx context.Context) ([]PresentRepo, error) {
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "github_id", "name_with_owner").
		From("repositories").
		Where(sq.Eq{"history_missing": false}).
		Where(sq.Lt{"star_count": 1_000_000}).
		OrderBy("star_count desc").
		OrderBy("id asc").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []PresentRepo
	for rows.Next() {
		var repo PresentRepo
		err := rows.Scan(&repo.Id, &repo.GithubId, &repo.NameWithOwner)
		if err != nil {
			return repos, err
		}
		repos = append(repos, repo)
	}

	if rows.Err() != nil {
		return repos, err
	}

	return repos, nil
}

func DeleteRepository(db *Database, ctx context.Context, id repoId) error {
	sql, args, err := sq.Delete("stars_history").
		Where(sq.Eq{"repository_id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = db.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	sql, args, err = sq.Delete("repositories").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = db.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}
