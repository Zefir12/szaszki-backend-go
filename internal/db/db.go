package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zefir/szaszki-go-backend/config"
)

var Pool *pgxpool.Pool

func Init() error {
	config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgURI := config.AppConfig.POSTGRES_URI // e.g. "postgres://user:pass@localhost:5432/dbname"

	// Create a connection pool
	var err error
	Pool, err = pgxpool.New(ctx, pgURI)
	if err != nil {
		return fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Ping to verify connection
	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	log.Println("PostgreSQL connection successfully established")

	// Ensure at least one table exists, e.g., create a dummy table if not exists
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS init_check (
		id SERIAL PRIMARY KEY,
		init BOOLEAN NOT NULL
	)`
	_, err = Pool.Exec(ctx, createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create init_check table: %w", err)
	}

	// Insert a dummy row to verify table insert
	_, err = Pool.Exec(ctx, "INSERT INTO init_check (init) VALUES (true)")
	if err != nil {
		return fmt.Errorf("failed to insert init row: %w", err)
	}

	// Optional: Clean up dummy row
	_, _ = Pool.Exec(ctx, "DELETE FROM init_check WHERE init = true")

	log.Println("Verified PostgreSQL database and table exist.")

	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
		log.Println("PostgreSQL connection pool closed")
	}
}
