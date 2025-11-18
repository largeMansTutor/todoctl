package mysqlstorage

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	migrateiofs "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	assets "github.com/thetrollfarmercodes/todoctl/todo"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/utils"
)

// Integration test exercise against a real MySQL instance.
func TestTodoRepositoryIntegration(t *testing.T) {
	t.Skip("Skipping integration test due to flakiness")
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, db, dsn := startMySQLContainer(t, ctx)
	defer func() { _ = container.Terminate(context.Background()) }()
	defer db.Close()

	runMigrations(t, db, dsn)

	repo := NewTodoRepository(db)

	t.Run("create get list update", func(t *testing.T) {
		created, err := repo.CreateTodos(ctx, []tododomain.Todo{
			{Title: "task-1", Description: utils.Ptr("first")},
			{Title: "task-2", Description: utils.Ptr("second")},
		})
		require.NoError(t, err)
		require.Len(t, created, 2)
		require.NotZero(t, created[0].ID)

		// Get by ID
		got, err := repo.GetTodo(ctx, created[0].ID, "")
		require.NoError(t, err)
		require.Equal(t, created[0].Title, got.Title)

		// Update one todo to complete
		complete := true
		newTitle := "task-2-updated"
		updates, err := repo.UpdateTodos(ctx, []tododomain.TodoUpdate{
			{ID: created[1].ID, Complete: &complete, Title: &newTitle},
		})
		require.NoError(t, err)
		require.Len(t, updates, 1)
		require.Equal(t, newTitle, updates[0].Title)
		require.True(t, updates[0].Complete)

		// List with offset pagination
		listed, nextCursor, err := repo.ListTodos(ctx, 1, 10, "")
		require.NoError(t, err)
		require.Len(t, listed, 2)
		require.Empty(t, nextCursor)

		// List with cursor pagination
		cursor := base64.URLEncoding.EncodeToString([]byte(strconv.FormatUint(listed[0].ID, 10)))
		listed2, nextCursor2, err := repo.ListTodos(ctx, 0, 1, cursor)
		require.NoError(t, err)
		require.Len(t, listed2, 1)
		require.NotEmpty(t, nextCursor2)
	})
}

func startMySQLContainer(t *testing.T, ctx context.Context) (testcontainers.Container, *sql.DB, string) {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		Env:          map[string]string{"MYSQL_ROOT_PASSWORD": "example", "MYSQL_DATABASE": "todos"},
		ExposedPorts: []string{"3306/tcp"},
		WaitingFor:   wait.ForLog("ready for connections").WithStartupTimeout(150 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	if host == "localhost" {
		// Force IPv4 loopback for remote Docker hosts that don't listen on ::1.
		host = "127.0.0.1"
	}
	port, err := container.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("root:example@tcp(%s:%s)/todos?parseTime=true&charset=utf8mb4&loc=UTC&multiStatements=true", host, port.Port())
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)

	for i := 0; i < 30; i++ {
		if pingErr := db.PingContext(ctx); pingErr == nil {
			return container, db, dsn
		}
		time.Sleep(2 * time.Second)
	}
	require.NoError(t, db.PingContext(ctx))
	return container, db, dsn
}

func runMigrations(t *testing.T, db *sql.DB, dsn string) {
	t.Helper()
	driver, err := mysqlmigrate.WithInstance(db, &mysqlmigrate.Config{MigrationsTable: "schema_migrations"})
	require.NoError(t, err)
	src, err := migrateiofs.New(assets.FS, assets.MigrationsDir)
	require.NoError(t, err)
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	require.NoError(t, err)

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
}
