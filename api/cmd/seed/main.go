// Command seed loads the development fixtures.
//
// The fixtures live next to the migrations and are idempotent, so the command can
// be run repeatedly against the same database:
//
//	cd api && go run ./cmd/seed
//
// When the API runs inside Docker the same file is available at /app/seeds.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gtrirf/start-and-found/api/internal/platform/config"
)

const defaultSeedFile = "seeds/dev.sql"

func main() {
	file := flag.String("file", defaultSeedFile, "SQL file to execute")
	flag.Parse()

	if err := run(*file); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run(file string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	script, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", file, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	connection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer connection.Release()

	// Executed through the simple protocol so the whole script - including its
	// BEGIN/COMMIT - runs as one batch.
	if _, err := connection.Conn().PgConn().Exec(ctx, string(script)).ReadAll(); err != nil {
		return fmt.Errorf("apply %s: %w", file, err)
	}

	fmt.Printf("seed: applied %s\n", file)
	return nil
}
