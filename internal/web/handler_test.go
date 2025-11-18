package web

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/idempotency"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/auth"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/service/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/service/todo/mocks"
	"go.uber.org/zap"
)

func TestCreateTodosHandler(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		idKey        string
		setupMocks   func(repo *mocks.Repository, store *mocks.IdempotencyStore)
		expectStatus int
		expectBody   string
	}{
		{
			name:    "creates todos and stores idempotency record",
			payload: `{"todos":[{"title":"x"}]}`,
			idKey:   "key",
			setupMocks: func(repo *mocks.Repository, store *mocks.IdempotencyStore) {
				items := []tododomain.Todo{{Title: "x"}}
				repo.On("CreateTodos", mock.Anything, items).Return(items, nil)
				store.On("Get", mock.Anything, "key", "todos:create").Return((*idempotency.Record)(nil), nil)
				store.On("Save", mock.Anything, "key", "todos:create", 201, mock.Anything).Return(nil)
			},
			expectStatus: http.StatusCreated,
			expectBody:   `"todos"`,
		},
		{
			name:    "replays cached response when idempotent",
			payload: `{"todos":[{"title":"ignored"}]}`,
			idKey:   "key",
			setupMocks: func(repo *mocks.Repository, store *mocks.IdempotencyStore) {
				store.On("Get", mock.Anything, "key", "todos:create").Return(&idempotency.Record{
					StatusCode: 201,
					Response:   []byte(`{"todos":[{"title":"cached"}]}`),
				}, nil)
			},
			expectStatus: http.StatusCreated,
			expectBody:   `{"todos":[{"title":"cached"}]}`,
		},
		{
			name:         "rejects invalid JSON",
			payload:      `{"todos":[{]`,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			store := mocks.NewIdempotencyStore(t)
			if tt.setupMocks != nil {
				tt.setupMocks(repo, store)
			}
			svc := todousecase.NewService(repo, store)
			authz := auth.New(map[string]string{tt.idKey: "test"})
			h := New(&config.Config{APIKey: tt.idKey}, svc, authz, nil, nil, zap.NewNop())

			req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader([]byte(tt.payload)))
			if tt.idKey != "" {
				req.Header.Set("Idempotency-Key", tt.idKey)
			}
			rec := httptest.NewRecorder()

			h.CreateTodos(rec, req)

			require.Equal(t, tt.expectStatus, rec.Code)
			if tt.expectBody != "" {
				if tt.expectBody[0] == '{' || tt.expectBody[0] == '[' {
					require.JSONEq(t, tt.expectBody, rec.Body.String())
				} else {
					require.Contains(t, rec.Body.String(), tt.expectBody)
				}
			}
			repo.AssertExpectations(t)
			store.AssertExpectations(t)
		})
	}
}

func TestUpdateTodosHandler(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		idKey        string
		setupMocks   func(repo *mocks.Repository, store *mocks.IdempotencyStore)
		expectStatus int
		expectBody   string
	}{
		{
			name:    "updates todos and stores idempotency record",
			payload: `{"todos":[{"id":1,"title":"x"}]}`,
			idKey:   "key",
			setupMocks: func(repo *mocks.Repository, store *mocks.IdempotencyStore) {
				updates := []tododomain.TodoUpdate{{ID: 1, Title: stringPtr("x")}}
				repo.On("UpdateTodos", mock.Anything, updates).Return([]tododomain.Todo{{ID: 1, Title: "x"}}, nil)
				store.On("Get", mock.Anything, "key", "todos:update").Return((*idempotency.Record)(nil), nil)
				store.On("Save", mock.Anything, "key", "todos:update", 200, mock.Anything).Return(nil)
			},
			expectStatus: http.StatusOK,
			expectBody:   `"todos"`,
		},
		{
			name:    "replays cached update response when idempotent",
			payload: `{"todos":[{"id":1,"title":"ignored"}]}`,
			idKey:   "key",
			setupMocks: func(repo *mocks.Repository, store *mocks.IdempotencyStore) {
				store.On("Get", mock.Anything, "key", "todos:update").Return(&idempotency.Record{
					StatusCode: 200,
					Response:   []byte(`{"todos":[{"id":1,"title":"cached"}]}`),
				}, nil)
			},
			expectStatus: http.StatusOK,
			expectBody:   `{"todos":[{"id":1,"title":"cached"}]}`,
		},
		{
			name:         "rejects invalid json",
			payload:      `{"todos":[{]}`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "rejects empty todos array",
			payload:      `{"todos":[]}`,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			store := mocks.NewIdempotencyStore(t)
			if tt.setupMocks != nil {
				tt.setupMocks(repo, store)
			}
			svc := todousecase.NewService(repo, store)
			authz := auth.New(map[string]string{tt.idKey: "test"})
			h := New(&config.Config{APIKey: tt.idKey}, svc, authz, nil, nil, zap.NewNop())

			req := httptest.NewRequest(http.MethodPatch, "/todos", bytes.NewReader([]byte(tt.payload)))
			if tt.idKey != "" {
				req.Header.Set("Idempotency-Key", tt.idKey)
			}
			rec := httptest.NewRecorder()

			h.UpdateTodos(rec, req)

			require.Equal(t, tt.expectStatus, rec.Code)
			if tt.expectBody != "" {
				if tt.expectBody[0] == '{' || tt.expectBody[0] == '[' {
					require.JSONEq(t, tt.expectBody, rec.Body.String())
				} else {
					require.Contains(t, rec.Body.String(), tt.expectBody)
				}
			}
			repo.AssertExpectations(t)
			store.AssertExpectations(t)
		})
	}
}

func TestListTodosHandler(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		setupMocks   func(repo *mocks.Repository)
		expectStatus int
		expectBody   string
	}{
		{
			name:  "lists todos with pagination defaults",
			query: "",
			setupMocks: func(repo *mocks.Repository) {
				repo.On("ListTodos", mock.Anything, 1, 10, "").Return([]tododomain.Todo{{ID: 1, Title: "a"}}, "", nil)
			},
			expectStatus: http.StatusOK,
			expectBody:   `"todos"`,
		},
		{
			name:  "uses cursor when provided",
			query: "?cursor=abc&limit=5",
			setupMocks: func(repo *mocks.Repository) {
				repo.On("ListTodos", mock.Anything, 1, 5, "abc").Return([]tododomain.Todo{}, "next", nil)
			},
			expectStatus: http.StatusOK,
			expectBody:   `"next_cursor"`,
		},
		{
			name:  "propagates errors to client",
			query: "",
			setupMocks: func(repo *mocks.Repository) {
				repo.On("ListTodos", mock.Anything, 1, 10, "").Return(nil, "", assert.AnError)
			},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			if tt.setupMocks != nil {
				tt.setupMocks(repo)
			}
			svc := todousecase.NewService(repo, nil)
			h := New(&config.Config{}, svc, nil, nil, nil, zap.NewNop())

			req := httptest.NewRequest(http.MethodGet, "/todos"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.ListTodos(rec, req)

			require.Equal(t, tt.expectStatus, rec.Code)
			if tt.expectBody != "" {
				require.Contains(t, rec.Body.String(), tt.expectBody)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestGetTodoHandler(t *testing.T) {
	repo := mocks.NewRepository(t)
	repo.On("GetTodo", mock.Anything, uint64(1), "").Return(&tododomain.Todo{ID: 1, Title: "x"}, nil)
	svc := todousecase.NewService(repo, nil)
	h := New(&config.Config{}, svc, nil, nil, nil, zap.NewNop())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req := httptest.NewRequest(http.MethodGet, "/todos/1", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	h.GetTodo(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"id":1`)
	repo.AssertExpectations(t)
}

func stringPtr[T any](v T) *T {
	return &v
}
