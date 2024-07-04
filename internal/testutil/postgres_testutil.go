package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glup3/TrendyGitHub/internal/db"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupPostgresContainer(ctx context.Context) (*db.Database, func(), error) {
	postgresContainer, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("tgh"),
		postgres.WithUsername("ditto"),
		postgres.WithPassword("blubblub"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	database, err := db.NewDatabase(ctx, connStr)
	if err != nil {
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("failed to find project root: %v", err)
	}

	migrationsPath := filepath.Join(projectRoot, "db", "migrations")
	m, err := migrate.New("file://"+migrationsPath, connStr)
	if err != nil {
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	cleanup := func() {
		database.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}

	return database, cleanup, nil
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
