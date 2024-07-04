package repository

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/internal/db"
)

type SettingsRepository struct {
	db  *db.Database
	ctx context.Context
}

type Settings struct {
	ID                     int
	CurrentMaxStarCount    int
	MinStarCount           int
	TimeoutSecondsPrevent  int
	TimeoutSecondsExceeded int
	TimeoutMaxUnits        int
	IsEnabled              bool
}

func NewSettingsRepository(ctx context.Context, db *db.Database) *SettingsRepository {
	return &SettingsRepository{
		db:  db,
		ctx: ctx,
	}
}

func (r *SettingsRepository) Load() (Settings, error) {
	var settings Settings

	sql, args, err := sq.
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
		Limit(1).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return settings, fmt.Errorf("error building SQL: %w", err)
	}

	err = r.db.Pool.QueryRow(r.ctx, sql, args...).
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
		return settings, fmt.Errorf("error loading settings: %w", err)
	}

	return settings, nil
}

func (r *SettingsRepository) UpdateStarCountCursor(newCount int, settingsID int) error {
	sql, args, err := sq.
		Update("settings").
		Set("current_max_star_count", newCount).
		Where(sq.Eq{"id": settingsID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("error building SQL: %v", err)
	}

	_, err = r.db.Pool.Exec(r.ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("error updating current max star count in SQL: %v", err)
	}

	return nil

}

func (r *SettingsRepository) ResetStarCountCursor(settingsID int) error {
	sql, args, err := sq.
		Update("settings").
		Set("current_max_star_count", 1_000_000).
		Where(sq.Eq{"id": settingsID}).
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
