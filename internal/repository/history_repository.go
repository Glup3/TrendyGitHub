package repository

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/internal/db"
	"github.com/jackc/pgx/v5"
)

type HistoryRepository struct {
	db  *db.Database
	ctx context.Context
}

type StarHistoryInput struct {
	CreatedAt time.Time
	StarCount int
	Id        int
}

func NewHistoryRepository(ctx context.Context, db *db.Database) *HistoryRepository {
	return &HistoryRepository{
		db:  db,
		ctx: ctx,
	}
}

func (r *HistoryRepository) BatchUpsertStarHistory(inputs []StarHistoryInput) error {
	const batchSize = 10_000

	tx, err := r.db.Pool.Begin(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(r.ctx)

	for start := 0; start < len(inputs); start += batchSize {
		end := start + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}

		query := sq.Insert("stars_history").Columns("repository_id", "star_count", "created_at")

		for _, input := range inputs[start:end] {
			query = query.Values(input.Id, input.StarCount, input.CreatedAt)
		}

		sql, args, err := query.
			Suffix(`
        ON CONFLICT (repository_id, created_at)
        DO UPDATE SET
        star_count = EXCLUDED.star_count,
        created_at = EXCLUDED.created_at
      `).
			PlaceholderFormat(sq.Dollar).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build SQL: %w", err)
		}

		if _, err := tx.Exec(r.ctx, sql, args...); err != nil {
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

	if _, err = tx.Exec(r.ctx, sql, args...); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	if err := tx.Commit(r.ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *HistoryRepository) CreateSnapshot() error {
	sql, args, err := sq.Insert("stars_history").
		Columns("repository_id", "star_count", "created_at").
		Select(
			sq.Select("id", "star_count", "CURRENT_DATE").From("repositories"),
		).
		Suffix(`
      ON CONFLICT (repository_id, created_at)
      DO UPDATE SET
      star_count = EXCLUDED.star_count
    `).
		ToSql()

	if err != nil {
		return fmt.Errorf("error building SQL: %w", err)
	}

	_, err = r.db.Pool.Exec(r.ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *HistoryRepository) RefreshView(view string) error {
	sqlStr := fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s", pgx.Identifier{view}.Sanitize())

	_, err := r.db.Pool.Exec(r.ctx, sqlStr)
	if err != nil {
		return err
	}

	return nil

}

func (r *HistoryRepository) DeleteForRepo(id int) error {
	sql, args, err := sq.
		Delete("stars_history").
		Where(sq.Eq{"repository_id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Pool.Exec(r.ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}