package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

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
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/service/todo"
	mysqlstorage2 "github.com/thetrollfarmercodes/todoctl/todo/internal/storage/mysql"
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
					loggerfx.New,
					metrics.New,
					telemetry.ConfigureTracerProvider,
					func(cfg *config.Config, logger *zap.Logger) *httpadapter.RateLimiter {
						return httpadapter.NewRateLimiter(cfg.RateLimitPerSecond, cfg.RateLimitBurst, logger)
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
					fx.Annotate(httpadapter.NewRouter, fx.ParamTags("", "", "", `group:"routes"`)),
				),
				fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, router http.Handler, logger *zap.Logger) {
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

			_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

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
