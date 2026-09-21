// Command migrate applies the embedded SQL migrations.
//
// Usage:
//
//	migrate up            apply every pending migration
//	migrate down [n]      roll back the last n migrations (default 1)
//	migrate version       print the current version
//	migrate force <ver>   mark the schema as version <ver> (repair a dirty state)
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"github.com/gtrirf/start-and-found/api/internal/platform/config"
	"github.com/gtrirf/start-and-found/api/migrations"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	switch command {
	case "up":
		err = migrator.Up()
	case "down":
		steps := 1
		if len(os.Args) > 2 {
			steps, err = strconv.Atoi(os.Args[2])
			if err != nil {
				return fmt.Errorf("down expects a number of steps: %w", err)
			}
		}
		err = migrator.Steps(-steps)
	case "version":
		return printVersion(migrator)
	case "force":
		return forceVersion(migrator, os.Args)
	default:
		return fmt.Errorf("unknown command %q (expected up, down, version or force)", command)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("migrations: already up to date")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Println("migrations: applied")
	return nil
}

func printVersion(migrator *migrate.Migrate) error {
	version, dirty, err := migrator.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("migrations: no version applied yet")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	fmt.Printf("migrations: version=%d dirty=%t\n", version, dirty)
	return nil
}

func forceVersion(migrator *migrate.Migrate, args []string) error {
	if len(args) < 3 {
		return errors.New("force expects a version")
	}
	version, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("force expects a numeric version: %w", err)
	}
	if err := migrator.Force(version); err != nil {
		return fmt.Errorf("force version: %w", err)
	}
	fmt.Printf("migrations: forced version=%d\n", version)
	return nil
}
