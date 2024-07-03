package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

var (
	dbInstance *Database
	dbOnce     sync.Once
	dbErr      error
)

func NewDatabase(ctx context.Context, connString string) (*Database, error) {
	dbOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)
		if err != nil {
			dbErr = fmt.Errorf("unable to create connection pool: %w", err)
			return
		}

		dbInstance = &Database{db}
	})

	if dbErr != nil {
		return nil, dbErr
	}

	return dbInstance, nil
}

func (db *Database) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func (db *Database) Close() {
	db.Pool.Close()
}
