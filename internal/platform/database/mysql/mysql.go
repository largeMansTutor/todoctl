package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/metrics"
)

// ProvideDB returns an Fx provider that opens a MySQL connection pool
// using the DSN from configuration. The returned *sql.DB is safe for concurrent
// use. If the database cannot be opened, application startup will
// fail. The DB is closed automatically when the Fx application stops.
func ProvideDB() fx.Option {
	type params struct {
		fx.In

		Lifecycle fx.Lifecycle
		Config    *config.Config
		Logger    *zap.Logger
		Metrics   *metrics.Provider `optional:"true"`
	}

	return fx.Provide(func(p params) (*sql.DB, error) {
		db, err := sql.Open("mysql", p.Config.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("mysql: open: %w", err)
		}
		if p.Metrics != nil {
			p.Metrics.RegisterDBMetrics(db)
		}
		// Set some sane limits. In production, these values should be
		// tuned based on expected concurrency and database capacity.
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(25)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Ping database on startup to ensure connection is healthy.
		p.Lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				if err := db.PingContext(ctx); err != nil {
					return fmt.Errorf("mysql: ping: %w", err)
				}
				p.Logger.Info("MySQL connection established")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				p.Logger.Info("Closing MySQL connection")
				return db.Close()
			},
		})
		return db, nil
	})
}

// MigrateUp applies filesystem migrations using golang-migrate against the
// provided DB handle. Applied versions are tracked in schema_migrations to
// prevent duplicate runs and ensure safe, repeatable deployments.
func MigrateUp(ctx context.Context, db *sql.DB, migrationsPath string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if migrationsPath == "" {
		return fmt.Errorf("migrationsPath is required")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	driver, err := mysqlmigrate.WithInstance(db, &mysqlmigrate.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("prepare migrate driver: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "mysql", driver)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
