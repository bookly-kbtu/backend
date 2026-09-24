package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/bookly-kbtu/backend/internal/bootstrap"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	command := os.Args[1]
	if !isAllowedCommand(command) {
		fmt.Fprintf(os.Stderr, "unknown migration command: %s\n\n", command)
		printUsage()
		os.Exit(2)
	}

	if err := run(command, os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "migrate %s: %v\n", command, err)
		os.Exit(1)
	}
}

func run(command string, args []string) error {
	cfg, err := bootstrap.LoadCommandConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("pgx", cfg.PostgresDSN())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err = goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.RunContext(context.Background(), command, db, cfg.MigrationsDir, args...)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/migrate <up|up-by-one|down|status|reset|redo|version|create> [goose args...]")
}

func isAllowedCommand(command string) bool {
	switch command {
	case "up", "up-by-one", "down", "status", "reset", "redo", "version", "create":
		return true
	default:
		return false
	}
}
