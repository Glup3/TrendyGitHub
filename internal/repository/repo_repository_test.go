package repository_test

import (
	"context"
	"testing"

	"github.com/glup3/TrendyGitHub/internal/repository"
	"github.com/glup3/TrendyGitHub/internal/testutil"
)

func TestRepoRepository_Test1(t *testing.T) {
	ctx := context.Background()

	db, cleanup, err := testutil.SetupPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("failed to set up test container: %v", err)
	}
	defer cleanup()

	r := repository.NewRepoRepository(ctx, db)

	_, err = r.FindNextMissing(100, repository.OrderAsc)
	if err == nil {
		t.Fatal("expected error")
	}
}
