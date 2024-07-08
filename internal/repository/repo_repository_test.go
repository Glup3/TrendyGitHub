package repository

import (
	"context"
	"testing"

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
}
