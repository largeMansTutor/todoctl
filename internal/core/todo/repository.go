package todo

import (
	"context"
	"errors"
)

// Repository defines the persistence boundary for Todo aggregates. It is
// implemented by storage adapters (e.g. MySQL) and consumed by
// the use case layer. Swapping the underlying database only requires
// providing another implementation of this interface.
type Repository interface {
	CreateTodos(ctx context.Context, items []Todo) ([]Todo, error)
	UpdateTodos(ctx context.Context, updates []TodoUpdate) ([]Todo, error)
	// ListTodos returns a slice of todos and a next cursor token. When the
	// cursor is non-empty it takes precedence over page/limit and enables
	// keyset pagination.
	ListTodos(ctx context.Context, page int, limit int, cursor string) ([]Todo, string, error)
}

// ErrDuplicateTitle indicates an attempt to create or update a todo with a
// title that already exists. The use case layer maps this error to the
// appropriate HTTP status without depending on storage-specific details.
var ErrDuplicateTitle = errors.New("duplicate title")
