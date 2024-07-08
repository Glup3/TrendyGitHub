package repository

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	database "github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepoRepository(t *testing.T) {
	connString, cleanup, restore, err := testutil.SetupPostgresContainer()
	if err != nil {
		t.Fatalf("failed to set up test container: %v", err)
	}
	defer cleanup()

	t.Run("Test finding next missing repo asc", func(t *testing.T) {
		t.Cleanup(func() {
			restore()
		})

		ctx := context.Background()
		pool, err := pgxpool.New(ctx, connString)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()

		r := NewRepoRepository(ctx, &database.Database{Pool: pool})

		repo, err := r.FindNextMissing(1_000_000, OrderAsc)
		if err != nil {
			t.Fatal(err)
		}

		if repo.Id != 1 {
			t.Fatalf("Expected %d to equal 1", repo.Id)
		}
	})

	t.Run("Test finding next missing desc", func(t *testing.T) {
		t.Cleanup(func() {
			restore()
		})

		ctx := context.Background()
		pool, err := pgxpool.New(ctx, connString)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()

		r := NewRepoRepository(ctx, &database.Database{Pool: pool})

		repo, err := r.FindNextMissing(1_000_000, OrderDesc)
		if err != nil {
			t.Fatal(err)
		}

		if repo.Id != 6 {
			t.Fatalf("Expected %d to equal 6", repo.Id)
		}
	})

	t.Run("Test upserting repos updates star count", func(t *testing.T) {
		t.Cleanup(func() {
			restore()
		})

		ctx := context.Background()
		pool, err := pgxpool.New(ctx, connString)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()

		r := NewRepoRepository(ctx, &database.Database{Pool: pool})
		repos := []RepoInput{
			{GithubId: "R_kg0001", Name: "glup3", NameWithOwner: "glup3/repo0001", StarCount: 922, ForkCount: 0, Languages: []string{}, PrimaryLanguage: "Go", Description: ""},
			{GithubId: "R_kg0002", Name: "glup3", NameWithOwner: "glup3/repo0002", StarCount: 300, ForkCount: 0, Languages: []string{}, PrimaryLanguage: "Go", Description: ""},
			{GithubId: "R_kg0003", Name: "glup3", NameWithOwner: "glup3/repo0003", StarCount: 410_233, ForkCount: 0, Languages: []string{}, PrimaryLanguage: "Go", Description: ""},
		}

		starCount, err := getStarCount(ctx, pool, "R_kg0003")
		if err != nil {
			t.Fatal(err)
		}
		if starCount != 400_000 {
			t.Fatalf("expected %d to equal 400000", starCount)
		}

		err = r.UpsertMany(repos)
		if err != nil {
			t.Fatal(err)
		}

		starCount, err = getStarCount(ctx, pool, "R_kg0003")
		if err != nil {
			t.Fatal(err)
		}
		if starCount != 410_233 {
			t.Fatalf("expected %d to equal 410233", starCount)
		}
	})
}

func getStarCount(ctx context.Context, pool *pgxpool.Pool, githubId string) (int, error) {
	var starCount int

	sql, args, err := sq.
		Select("star_count").
		From("repositories").
		Where(sq.Eq{"github_id": githubId}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return starCount, err
	}

	err = pool.QueryRow(ctx, sql, args...).Scan(&starCount)
	if err != nil {
		return starCount, err
	}

	return starCount, nil
}
