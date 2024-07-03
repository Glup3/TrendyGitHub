package repository

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/internal/db"
	"github.com/jackc/pgx/v5"
)

type HistoryRepository struct {
	db  *db.Database
	ctx context.Context
}

func NewHistoryRepository(db *db.Database, ctx context.Context) *HistoryRepository {
	return &HistoryRepository{
		db:  db,
		ctx: ctx,
	}
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
