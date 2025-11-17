package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	mysqlstorage "github.com/thetrollfarmercodes/todoctl/todo/internal/repository/mysql"
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/service/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
)

// End-to-end HTTP test against a real MySQL using testcontainers.
func TestHTTPIntegration(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container, db, dsn := startMySQL(t, ctx)
	defer func() { _ = container.Terminate(context.Background()) }()
	defer db.Close()

	runMigrations(t, db, dsn)

	cfg := &config.Config{
		HTTPConfig: httpadapter.HTTPConfig{
			HTTPPort:     0,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  10 * time.Second,
			MaxBodyBytes: 1 << 20,
			OTELConfig: httpadapter.OTELConfig{
				OTELServiceName: "todo-test",
			},
		},
		APIKey: "dev-key",
	}

	repo := mysqlstorage.NewTodoRepository(db)
	idStore := mysqlstorage.NewIdempotencyStore(db)
	svc := todousecase.NewService(repo, idStore)
	app := New(cfg, svc, httpadapter.NewRateLimiter(100, 100, nil), nil, zap.NewNop())

	router, err := httpadapter.NewRouter(&cfg.HTTPConfig, zap.NewNop(), nil, app)
	require.NoError(t, err)
	server := httptest.NewServer(router)
	defer server.Close()

	type todoResp struct {
		Todos []map[string]interface{} `json:"todos"`
		Todo  map[string]interface{}   `json:"todo"`
	}

	client := server.Client()

	t.Run("create and get", func(t *testing.T) {
		payload := `{"todos":[{"title":"first","description":"one"}]}`
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/todos", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", cfg.APIKey)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var created todoResp
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
		require.Len(t, created.Todos, 1)
		id := int(created.Todos[0]["id"].(float64))

		reqGet, _ := http.NewRequest(http.MethodGet, server.URL+"/todos/"+strconv.Itoa(id), nil)
		reqGet.Header.Set("X-API-Key", cfg.APIKey)
		respGet, err := client.Do(reqGet)
		require.NoError(t, err)
		defer respGet.Body.Close()
		require.Equal(t, http.StatusOK, respGet.StatusCode)
	})

	t.Run("idempotent create replays", func(t *testing.T) {
		payload := `{"todos":[{"title":"second"}]}`
		key := "idem-1"
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/todos", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", cfg.APIKey)
		req.Header.Set("Idempotency-Key", key)

		resp1, err := client.Do(req)
		require.NoError(t, err)
		defer resp1.Body.Close()
		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		resp2, err := client.Do(req)
		require.NoError(t, err)
		defer resp2.Body.Close()
		require.Equal(t, http.StatusCreated, resp2.StatusCode)
	})
}

func startMySQL(t *testing.T, ctx context.Context) (testcontainers.Container, *sql.DB, string) {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		Env:          map[string]string{"MYSQL_ROOT_PASSWORD": "example", "MYSQL_DATABASE": "todos"},
		ExposedPorts: []string{"3306/tcp"},
		WaitingFor:   wait.ForLog("ready for connections").WithStartupTimeout(90 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("root:example@tcp(%s:%s)/todos?parseTime=true&charset=utf8mb4&loc=UTC", host, port.Port())
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		if err := db.PingContext(ctx); err == nil {
			return container, db, dsn
		}
		time.Sleep(time.Second)
	}
	require.NoError(t, db.PingContext(ctx))
	return container, db, dsn
}

func runMigrations(t *testing.T, db *sql.DB, dsn string) {
	t.Helper()
	driver, err := mysqlmigrate.WithInstance(db, &mysqlmigrate.Config{MigrationsTable: "schema_migrations"})
	require.NoError(t, err)
	sourceURL := fmt.Sprintf("file://%s", filepath.Join("..", "..", "..", "migrations"))
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "mysql", driver)
	require.NoError(t, err)
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}
