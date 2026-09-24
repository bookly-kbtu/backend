package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bookly-kbtu/backend/internal/bootstrap"
	"github.com/bookly-kbtu/backend/internal/transport/command"
)

// CLI entry point for maintenance jobs: importing external catalogues, etc.
// Usage: go run ./cmd/command <command> [flags]
func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := bootstrap.LoadCommandConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		return 1
	}

	deps, err := bootstrap.NewCommandDeps(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init deps:", err)
		return 1
	}
	defer deps.Close()

	return command.New(deps.Importer, os.Stdout, os.Stderr).Run(ctx, os.Args[1:])
}
