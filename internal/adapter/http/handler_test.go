package httpadapter

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/idempotency"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/usecase/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/usecase/todo/mocks"
)

func TestCreateTodosHandlerSuccess(t *testing.T) {
	repo := mocks.NewRepository(t)
	idStore := mocks.NewIdempotencyStore(t)
	items := []tododomain.Todo{{Title: "x"}}
	repo.On("CreateTodos", mock.Anything, items).Return(items, nil)
	idStore.On("Get", mock.Anything, "key", "todos:create").Return((*idempotency.Record)(nil), nil)
	idStore.On("Save", mock.Anything, "key", "todos:create", 201, mock.Anything).Return(nil)
	svc := todousecase.NewService(repo, idStore)
	h := NewHandler(svc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader([]byte(`{"todos":[{"title":"x"}]}`)))
	req.Header.Set("Idempotency-Key", "key")
	rec := httptest.NewRecorder()

	h.CreateTodos(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"todos"`)
	repo.AssertExpectations(t)
	idStore.AssertExpectations(t)
}

func TestCreateTodosHandlerReplay(t *testing.T) {
	repo := mocks.NewRepository(t)
	idStore := mocks.NewIdempotencyStore(t)
	idStore.On("Get", mock.Anything, "key", "todos:create").Return(&idempotency.Record{
		StatusCode: 201,
		Response:   []byte(`{"todos":[{"title":"cached"}]}`),
	}, nil)
	svc := todousecase.NewService(repo, idStore)
	h := NewHandler(svc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewReader([]byte(`{"todos":[{"title":"ignored"}]}`)))
	req.Header.Set("Idempotency-Key", "key")
	rec := httptest.NewRecorder()

	h.CreateTodos(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.JSONEq(t, `{"todos":[{"title":"cached"}]}`, rec.Body.String())
	repo.AssertExpectations(t)
	idStore.AssertExpectations(t)
}
