package repository

import (
	"context"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/glup3/TrendyGitHub/internal/db"
	"github.com/glup3/TrendyGitHub/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHistoryRepository(t *testing.T) {
	connString, cleanup, restore, err := testutil.SetupPostgresContainer()
	if err != nil {
		t.Fatalf("failed to set up test container: %v", err)
	}
	defer cleanup()

	t.Run("Test creating snapshot overrides star count", func(t *testing.T) {
		t.Cleanup(func() {
			restore()
		})

		ctx := context.Background()
		pool, err := pgxpool.New(ctx, connString)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()

		hRepo := NewHistoryRepository(ctx, &db.Database{Pool: pool})
		rRepo := NewRepoRepository(ctx, &db.Database{Pool: pool})

		err = hRepo.CreateSnapshot()
		if err != nil {
			t.Fatal(err)
		}

		repoId := 1

		starCount, err := rRepo.GetStarCount(repoId, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if starCount != 200 {
			t.Fatalf("Expected %d to equal 200", starCount)
		}

		sql, args, err := sq.
			Update("repositories").
			Set("star_count", 210).
			Where(sq.Eq{"id": repoId}).
			PlaceholderFormat(sq.Dollar).
			ToSql()
		if err != nil {
			t.Fatal(err)
		}

		_, err = pool.Exec(ctx, sql, args...)
		if err != nil {
			t.Fatal(err)
		}

		err = hRepo.CreateSnapshot()
		if err != nil {
			t.Fatal(err)
		}

		starCount, err = rRepo.GetStarCount(repoId, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if starCount != 210 {
			t.Fatalf("Expected %d to equal 210", starCount)
		}

		sql, args, err = sq.
			Update("repositories").
			Set("star_count", 500).
			Where(sq.Eq{"id": repoId}).
			PlaceholderFormat(sq.Dollar).
			ToSql()
		if err != nil {
			t.Fatal(err)
		}

		_, err = pool.Exec(ctx, sql, args...)
		if err != nil {
			t.Fatal(err)
		}

		err = hRepo.CreateSnapshot()
		if err != nil {
			t.Fatal(err)
		}

		starCount, err = rRepo.GetStarCount(repoId, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if starCount != 500 {
			t.Fatalf("Expected %d to equal 500", starCount)
		}
	})
}
