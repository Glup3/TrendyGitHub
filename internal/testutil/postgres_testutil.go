package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupPostgresContainer() (string, func(), func(), error) {
	ctx := context.Background()
	container, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("tgh"),
		postgres.WithUsername("ditto"),
		postgres.WithPassword("blubblub"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to start container: %w", err)
	}

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	migrationsPath := filepath.Join(projectRoot, "db", "migrations")
	m, err := migrate.New("file://"+migrationsPath, connString)
	if err != nil {
		container.Terminate(ctx)
		return "", nil, nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		container.Terminate(ctx)
		return "", nil, nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	err = seedDatabase(ctx, connString)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to seed database: %w", err)
	}

	srcErr, dbErr := m.Close()
	if srcErr != nil {
		return "", nil, nil, srcErr
	}
	if dbErr != nil {
		return "", nil, nil, dbErr
	}

	err = container.Snapshot(ctx, postgres.WithSnapshotName("test-snapshot"))
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to take snapshot: %w", err)
	}

	connString, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	restore := func() {
		err := container.Restore(ctx)
		if err != nil {
			log.Fatalf("failed to restore container: %s", err)
		}
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}

	return connString, cleanup, restore, nil
}

func seedDatabase(ctx context.Context, connString string) error {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return err
	}
	defer pool.Close()

	sql, args, err := sq.
		Insert("repositories").
		Columns("id", "github_id", "name", "name_with_owner", "star_count", "fork_count", "primary_language", "history_missing", "languages").
		Values(0001, "R_kg0001", "glup3", "glup3/repo0001", 200, 0, "Go", true, []string{}).
		Values(0002, "R_kg0002", "glup3", "glup3/repo0002", 400, 0, "Go", true, []string{}).
		Values(0003, "R_kg0003", "glup3", "glup3/repo0003", 400_000, 0, "Go", false, []string{}).
		Values(0004, "R_kg0004", "glup3", "glup3/repo0004", 30_000, 0, "Go", true, []string{}).
		Values(0005, "R_kg0005", "glup3", "glup3/repo0005", 1000, 0, "Go", true, []string{}).
		Values(0006, "R_kg0006", "glup3", "glup3/repo0006", 84_000, 0, "Go", true, []string{}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}

	return "", fmt.Errorf("project root not found")
}
