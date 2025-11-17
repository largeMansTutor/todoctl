package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

// MigrateUp executes all migration statements within the provided
// statements slice. Each statement should be idempotent. Errors halt
// execution. This is a simplistic migration helper and intentionally
// conservative: it does not handle rolling back partially applied
// migrations. In production use a real migration tool.
func MigrateUp(ctx context.Context, db *sql.DB, statements []string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() // rollback will be a no-op if commit was successful
	}()
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w; statement: %s", err, stmt)
		}
	}
	return tx.Commit()
}
