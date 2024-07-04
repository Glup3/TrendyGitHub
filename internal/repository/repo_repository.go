package repository

import (
	"context"
	"fmt"

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

func (r *RepoRepository) GetAllPresentHistoryRepos() ([]Repo, error) {
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

	rows, err := r.db.Pool.Query(r.ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var repo Repo
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
