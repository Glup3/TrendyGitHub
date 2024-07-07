package repository

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/internal/db"
)

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

type RepoRepository struct {
	db  *db.Database
	ctx context.Context
}

type Repo struct {
	GithubId      string
	NameWithOwner string
	StarCount     int
	Id            int
}

type BrokenRepo struct {
	Id            int
	StarCount     int
	UntilDate     time.Time
	NameWithOwner string
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

type SortOrder string

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

func (r *RepoRepository) FindNextMissing(maxStarCount int, order SortOrder) (Repo, error) {
	var repo Repo

	if order != OrderAsc && order != OrderDesc {
		return repo, fmt.Errorf("invalid sort order %s", order)
	}

	sql, args, err := sq.
		Select("id", "github_id", "star_count", "name_with_owner").
		From("repositories").
		Where(sq.Eq{"history_missing": true}).
		Where(sq.LtOrEq{"star_count": maxStarCount}).
		OrderBy("star_count " + string(order)).
		Limit(1).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return repo, err
	}

	err = r.db.Pool.QueryRow(r.ctx, sql, args...).Scan(&repo.Id, &repo.GithubId, &repo.StarCount, &repo.NameWithOwner)
	if err != nil {
		return repo, err
	}

	return repo, nil
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

func (r *RepoRepository) Delete(id int) error {
	sql, args, err := sq.
		Delete("repositories").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	_, err = r.db.Pool.Exec(r.ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *RepoRepository) MarkAsDone(id int) error {
	sql, args, err := sq.
		Update("repositories").
		Set("history_missing", false).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	_, err = r.db.Pool.Exec(r.ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *RepoRepository) GetStarCount(id int, date time.Time) (int, error) {
	var starCount int

	sql, args, err := sq.
		Select("star_count").
		From("stars_history").
		Where(sq.Eq{"repository_id": id}).
		Where(sq.Eq{"created_at": date}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return starCount, fmt.Errorf("failed to build SQL: %w", err)
	}

	err = r.db.Pool.QueryRow(r.ctx, sql, args...).Scan(&starCount)
	if err != nil {
		return starCount, err
	}

	return starCount, nil
}
