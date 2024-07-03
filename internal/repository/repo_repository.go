package repository

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/internal/db"
)

type RepoRepository struct {
	db  *db.Database
	ctx context.Context
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

type LanguageInput struct {
	Id       string
	Hexcolor string
}

func NewRepoRepository(ctx context.Context, db *db.Database) *RepoRepository {
	return &RepoRepository{
		db:  db,
		ctx: ctx,
	}
}

func (r *RepoRepository) UpsertMany(repos []RepoInput) error {
	query := sq.Insert("repositories").
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
		query = query.Values(
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

	sql, args, err := query.
		Suffix(`
			ON CONFLICT (github_id)
			DO UPDATE SET
				star_count = EXCLUDED.star_count,
				fork_count = EXCLUDED.fork_count,
        primary_language = EXCLUDED.primary_language,
				languages = EXCLUDED.languages,
        description = EXCLUDED.description
		`).
		PlaceholderFormat(sq.Dollar).
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

func (r *RepoRepository) UpsertLanguages(languages []LanguageInput) error {
	if len(languages) == 0 {
		return nil
	}

	query := sq.Insert("languages").Columns("id", "hexcolor")

	for _, lang := range languages {
		query = query.Values(lang.Id, lang.Hexcolor)
	}

	sql, args, err := query.
		Suffix(`
			ON CONFLICT (id)
			DO UPDATE SET hexcolor = EXCLUDED.hexcolor
		`).
		PlaceholderFormat(sq.Dollar).
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
