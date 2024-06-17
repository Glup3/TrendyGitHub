package main

import (
	"context"
	"log"

	config "github.com/glup3/TrendyGitHub/internal"
	database "github.com/glup3/TrendyGitHub/internal/db"
)

func main() {
	ctx := context.Background()

	configs, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	db, err := database.NewDatabase(ctx, configs.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	err = db.Ping(ctx)
	if err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	var errors []error

	err = database.RefreshHistoryView(db, ctx, "mv_daily_stars")
	if err != nil {
		errors = append(errors, err)
	}

	err = database.RefreshHistoryView(db, ctx, "mv_weekly_stars")
	if err != nil {
		errors = append(errors, err)
	}

	err = database.RefreshHistoryView(db, ctx, "mv_monthly_stars")
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		log.Println(errors)
	}

	// rows, err := database.CreateSnapshotAndReset(db, ctx, 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// log.Println("Created snapshot with repo count", rows)
}
