package todousecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/idempotency"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/usecase/todo/mocks"
)

func TestCreateTodos(t *testing.T) {
	tests := []struct {
		name          string
		input         []tododomain.Todo
		setupStore    func(store *mocks.IdempotencyStore)
		repoErr       error
		expectErr     string
		expectCreated int
		expectReplay  bool
	}{
		{
			name:      "empty title fails",
			input:     []tododomain.Todo{{Title: ""}},
			expectErr: "title cannot be empty",
		},
		{
			name:      "duplicate titles in batch fails",
			input:     []tododomain.Todo{{Title: "a"}, {Title: "a"}},
			expectErr: "duplicate title in batch",
		},
		{
			name:  "idempotent replay returns stored response",
			input: []tododomain.Todo{{Title: "a"}},
			setupStore: func(store *mocks.IdempotencyStore) {
				store.On("Get", mock.Anything, "key", "todos:create").Return(&idempotency.Record{
					StatusCode: 201,
					Response:   []byte(`{"todos":[{"title":"a"}]}`),
				}, nil)
			},
			expectReplay: true,
		},
		{
			name:      "duplicate title from repo returns conflict error",
			input:     []tododomain.Todo{{Title: "a"}},
			repoErr:   tododomain.ErrDuplicateTitle,
			expectErr: "title conflict",
		},
		{
			name:          "success stores and returns created todos",
			input:         []tododomain.Todo{{Title: "a"}},
			expectCreated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			idStore := mocks.NewIdempotencyStore(t)
			repo.On("CreateTodos", mock.Anything, tt.input).Return(tt.input, tt.repoErr).Maybe()
			idStore.On("Get", mock.Anything, "key", "todos:create").Return((*idempotency.Record)(nil), nil).Maybe()
			idStore.On("Save", mock.Anything, "key", "todos:create", 201, mock.Anything).Return(nil).Maybe()
			if tt.setupStore != nil {
				tt.setupStore(idStore)
			}
			svc := NewService(repo, idStore)

			created, replay, err := svc.CreateTodos(context.Background(), "key", "todos:create", tt.input)

			if tt.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
				repo.AssertExpectations(t)
				idStore.AssertExpectations(t)
				return
			}
			require.NoError(t, err)
			if tt.expectReplay {
				assert.Nil(t, created)
				assert.NotNil(t, replay)
				idStore.AssertNotCalled(t, "Save", mock.Anything, "key", "todos:create", mock.Anything, mock.Anything)
				repo.AssertExpectations(t)
				idStore.AssertExpectations(t)
				return
			}
			require.Len(t, created, tt.expectCreated)
			idStore.AssertCalled(t, "Save", mock.Anything, "key", "todos:create", 201, mock.Anything)
			repo.AssertExpectations(t)
			idStore.AssertExpectations(t)
		})
	}
}

func TestUpdateTodos(t *testing.T) {
	tests := []struct {
		name         string
		input        []tododomain.TodoUpdate
		setupStore   func(store *mocks.IdempotencyStore)
		repoErr      error
		expectErr    string
		expectReplay bool
		expectCount  int
	}{
		{
			name:      "missing id fails",
			input:     []tododomain.TodoUpdate{{Title: stringPtr("a")}},
			expectErr: "id is required",
		},
		{
			name: "duplicate ids in batch fails",
			input: []tododomain.TodoUpdate{
				{ID: 1, Title: stringPtr("x")},
				{ID: 1, Title: stringPtr("y")},
			},
			expectErr: "duplicate id in batch",
		},
		{
			name:  "idempotent replay returns stored response",
			input: []tododomain.TodoUpdate{{ID: 1, Title: stringPtr("x")}},
			setupStore: func(store *mocks.IdempotencyStore) {
				store.On("Get", mock.Anything, "key", "todos:update").Return(&idempotency.Record{
					StatusCode: 200,
					Response:   []byte(`{"todos":[{"id":1}]}`),
				}, nil)
			},
			expectReplay: true,
		},
		{
			name:      "duplicate title from repo surfaces conflict",
			input:     []tododomain.TodoUpdate{{ID: 2, Title: stringPtr("dup")}},
			repoErr:   tododomain.ErrDuplicateTitle,
			expectErr: "title conflict",
		},
		{
			name:        "success updates todos",
			input:       []tododomain.TodoUpdate{{ID: 2, Title: stringPtr("ok")}},
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewRepository(t)
			repo.On("UpdateTodos", mock.Anything, tt.input).Return([]tododomain.Todo{{ID: 2}}, tt.repoErr).Maybe()
			idStore := mocks.NewIdempotencyStore(t)
			idStore.On("Get", mock.Anything, "key", "todos:update").Return((*idempotency.Record)(nil), nil).Maybe()
			idStore.On("Save", mock.Anything, "key", "todos:update", 200, mock.Anything).Return(nil).Maybe()
			if tt.setupStore != nil {
				tt.setupStore(idStore)
			}
			svc := NewService(repo, idStore)

			updated, replay, err := svc.UpdateTodos(context.Background(), "key", "todos:update", tt.input)

			if tt.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
				repo.AssertExpectations(t)
				idStore.AssertExpectations(t)
				return
			}
			require.NoError(t, err)
			if tt.expectReplay {
				assert.Nil(t, updated)
				assert.NotNil(t, replay)
				idStore.AssertNotCalled(t, "Save", mock.Anything, "key", "todos:update", mock.Anything, mock.Anything)
				repo.AssertExpectations(t)
				idStore.AssertExpectations(t)
				return
			}
			require.Len(t, updated, tt.expectCount)
			idStore.AssertCalled(t, "Save", mock.Anything, "key", "todos:update", 200, mock.Anything)
			repo.AssertExpectations(t)
			idStore.AssertExpectations(t)
		})
	}
}

func stringPtr[T any](i T) *T { return &i }
