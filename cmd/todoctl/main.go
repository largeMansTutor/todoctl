package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/idempotency"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	mysqlfx "github.com/thetrollfarmercodes/todoctl/todo/internal/platform/database/mysql"
	loggerfx "github.com/thetrollfarmercodes/todoctl/todo/internal/platform/logger"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/metrics"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/telemetry"
	mysqlstorage2 "github.com/thetrollfarmercodes/todoctl/todo/internal/repository/mysql"
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/service/todo"
	webapi "github.com/thetrollfarmercodes/todoctl/todo/internal/web"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	migrateDirection string
	migrateSteps     int
	migrationPath    string
)

func main() {
	root := &cobra.Command{
		Use:   "todoctl",
		Short: "Todo service CLI",
	}

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Run the HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := fx.New(
				fx.Provide(
					config.New,
					func(cfg *config.Config) *httpadapter.HTTPConfig {
						return &cfg.HTTPConfig
					},
					loggerfx.New,
					metrics.New,
					telemetry.ConfigureTracerProvider,
					func(cfg *config.Config, logger *zap.Logger) *httpadapter.RateLimiter {
						return httpadapter.NewRateLimiter(cfg.RateLimitPerSecond, cfg.RateLimitBurst, logger)
					},
					func(p *metrics.Provider) httpadapter.MetricsProvider {
						return p
					},
				),
				mysqlfx.ProvideDB(),
				fx.Provide(
					func(db *sql.DB) tododomain.Repository {
						return mysqlstorage2.NewTodoRepository(db)
					},
					func(db *sql.DB) idempotency.Store {
						return mysqlstorage2.NewIdempotencyStore(db)
					},
					todousecase.NewService,
					fx.Annotate(webapi.New, fx.As(new(httpadapter.RouteMounter)), fx.ResultTags(`group:"routes"`)),
					fx.Annotate(httpadapter.NewRouter, fx.ParamTags("", "", `optional:"true"`, `group:"routes"`)),
				),
				fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, router chi.Router, logger *zap.Logger, metrics *metrics.Provider) {
					httpadapter.StartHTTPServer(lc, &cfg.HTTPConfig, router, logger)
				}),
			)
			if err := app.Start(context.Background()); err != nil {
				return err
			}
			<-app.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return app.Stop(ctx)
		},
	}

	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations (up/down)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New()
			if err != nil {
				return err
			}
			db, err := sql.Open("mysql", cfg.DatabaseDSN)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := waitForDB(cfg.DatabaseDSN, 8, 3*time.Second); err != nil {
				return fmt.Errorf("database not ready: %w", err)
			}

			driver, err := mysqlmigrate.WithInstance(db, &mysqlmigrate.Config{
				MigrationsTable: "schema_migrations",
			})
			if err != nil {
				return err
			}
			sourceURL := fmt.Sprintf("file://%s", migrationPath)
			m, err := migrate.NewWithDatabaseInstance(sourceURL, "mysql", driver)
			if err != nil {
				return err
			}
			switch migrateDirection {
			case "up":
				if migrateSteps > 0 {
					err = m.Steps(migrateSteps)
				} else {
					err = m.Up()
				}
			case "down":
				if migrateSteps > 0 {
					err = m.Steps(-migrateSteps)
				} else {
					err = m.Down()
				}
			default:
				return fmt.Errorf("invalid direction %q (use up|down)", migrateDirection)
			}
			if err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return err
			}
			fmt.Println("Migrations applied successfully")
			return nil
		},
	}

	migrateCmd.Flags().StringVar(&migrateDirection, "direction", "up", "migration direction (up|down)")
	migrateCmd.Flags().IntVar(&migrateSteps, "steps", 0, "number of steps to migrate (0 = all pending)")
	migrateCmd.Flags().StringVar(&migrationPath, "path", "migrations", "path to migration files")

	root.AddCommand(apiCmd)
	root.AddCommand(migrateCmd)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// waitForDB pings the DB with simple backoff; useful in containerized starts
// where MySQL may still be initializing when migrations run.
func waitForDB(dsn string, attempts int, delay time.Duration) error {
	if attempts < 1 {
		attempts = 1
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(math.Pow(1.3, float64(i))) * delay)
	}
	return err
}
