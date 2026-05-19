package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// runMigrate handles the `migrate` subcommand of the agentsmesh-backend
// binary. The migration SQL lives inside the binary (//go:embed *.sql in
// //backend/migrations), so deployment only needs the binary itself —
// no companion `migrate` CLI, no /app/migrations mount.
//
// Invocations:
//
//	agentsmesh-backend migrate up           # apply all pending
//	agentsmesh-backend migrate up <N>       # apply at most N
//	agentsmesh-backend migrate down <N>     # rollback N (default 1)
//	agentsmesh-backend migrate version      # print current version + dirty flag
//	agentsmesh-backend migrate force <V>    # force schema_migrations.version (recover from dirty)
//
// Database connection params come from the same env vars the server
// boot path uses (DATABASE_URL or POSTGRES_HOST/POSTGRES_USER/...), so
// migrate.sh just sources the env file and invokes the subcommand.
func runMigrate(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open embedded migration source: %v\n", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL(cfg.Database))
	if err != nil {
		fmt.Fprintf(os.Stderr, "init migrate: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	sub := "up"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "up":
		if len(args) > 1 {
			n, parseErr := strconv.Atoi(args[1])
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "migrate up <N>: %v\n", parseErr)
				os.Exit(1)
			}
			err = m.Steps(n)
		} else {
			err = m.Up()
		}
	case "down":
		n := 1
		if len(args) > 1 {
			parsed, parseErr := strconv.Atoi(args[1])
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "migrate down <N>: %v\n", parseErr)
				os.Exit(1)
			}
			n = parsed
		}
		err = m.Steps(-n)
	case "version":
		v, dirty, vErr := m.Version()
		if errors.Is(vErr, migrate.ErrNilVersion) {
			fmt.Println("nil")
			return
		}
		if vErr != nil {
			fmt.Fprintf(os.Stderr, "version: %v\n", vErr)
			os.Exit(1)
		}
		fmt.Printf("%d (dirty=%v)\n", v, dirty)
		return
	case "force":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: migrate force <version>")
			os.Exit(1)
		}
		v, parseErr := strconv.Atoi(args[1])
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "migrate force <V>: %v\n", parseErr)
			os.Exit(1)
		}
		err = m.Force(v)
	default:
		fmt.Fprintf(os.Stderr, "unknown migrate subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "usage: migrate {up [N] | down [N] | version | force <V>}")
		os.Exit(1)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Fprintf(os.Stderr, "migrate %s: %v\n", sub, err)
		os.Exit(1)
	}
	slog.Info("migrate ok", "subcommand", sub)
}

func migrateURL(c config.DatabaseConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.DBName,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
