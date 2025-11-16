package todousecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/idempotency"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
)

// Service provides business logic for creating, updating and listing
// todos. It wraps a Repository and an IdempotencyStore to provide
// idempotent operations.
type Service struct {
	repo    tododomain.Repository
	idStore idempotency.Store
}

// NewService constructs a Service with the given repository and
// idempotency store. Either may be nil in tests.
func NewService(repo tododomain.Repository, idStore idempotency.Store) *Service {
	return &Service{
		repo:    repo,
		idStore: idStore,
	}
}

// CreateTodos processes a batch create request. If idempotencyKey is
// non-empty, it first checks for a stored response and returns it. On
// success it stores the response body in the idempotency store. It
// returns the created todos and possibly an encoded JSON response for
// storage.
func (s *Service) CreateTodos(ctx context.Context, idempotencyKey, scope string, items []tododomain.Todo) ([]tododomain.Todo, []byte, error) {
	if idempotencyKey != "" && s.idStore != nil {
		rec, err := s.idStore.Get(ctx, idempotencyKey, scope)
		if err != nil {
			return nil, nil, err
		}
		if rec != nil {
			// Replay previous response
			return nil, rec.Response, nil
		}
	}
	// Validate titles are not empty and unique within the batch
	seen := map[string]struct{}{}
	for _, t := range items {
		if t.Title == "" {
			return nil, nil, fmt.Errorf("title cannot be empty")
		}
		if _, exists := seen[t.Title]; exists {
			return nil, nil, fmt.Errorf("duplicate title in batch: %s", t.Title)
		}
		seen[t.Title] = struct{}{}
	}
	created, err := s.repo.CreateTodos(ctx, items)
	if err != nil {
		if errors.Is(err, tododomain.ErrDuplicateTitle) {
			return nil, nil, fmt.Errorf("title conflict: %w", err)
		}
		return nil, nil, err
	}
	// Marshal response for storage. We include the wrapper to match API output.
	body, _ := json.Marshal(map[string]interface{}{
		"todos": created,
	})
	if idempotencyKey != "" && s.idStore != nil {
		// Save result; ignore errors (best effort)
		_ = s.idStore.Save(ctx, idempotencyKey, scope, 201, body)
	}
	return created, body, nil
}

// UpdateTodos processes partial updates. It supports idempotency like
// CreateTodos. It returns updated todos and JSON body for storage.
func (s *Service) UpdateTodos(ctx context.Context, idempotencyKey, scope string, updates []tododomain.TodoUpdate) ([]tododomain.Todo, []byte, error) {
	if idempotencyKey != "" && s.idStore != nil {
		rec, err := s.idStore.Get(ctx, idempotencyKey, scope)
		if err != nil {
			return nil, nil, err
		}
		if rec != nil {
			return nil, rec.Response, nil
		}
	}
	// Validate: each update must have ID; ensure no duplicate IDs
	seen := map[uint64]struct{}{}
	for _, u := range updates {
		if u.ID == 0 {
			return nil, nil, fmt.Errorf("id is required")
		}
		if _, ok := seen[u.ID]; ok {
			return nil, nil, fmt.Errorf("duplicate id in batch: %d", u.ID)
		}
		seen[u.ID] = struct{}{}
	}
	updated, err := s.repo.UpdateTodos(ctx, updates)
	if err != nil {
		if errors.Is(err, tododomain.ErrDuplicateTitle) {
			return nil, nil, fmt.Errorf("title conflict: %w", err)
		}
		return nil, nil, err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"todos": updated,
	})
	if idempotencyKey != "" && s.idStore != nil {
		_ = s.idStore.Save(ctx, idempotencyKey, scope, 200, body)
	}
	return updated, body, nil
}

// ListTodos lists todos using pagination. It does not consider
// idempotency since it is a read-only operation. It returns next
// cursor for keyset pagination.
func (s *Service) ListTodos(ctx context.Context, page int, limit int, cursor string) ([]tododomain.Todo, string, error) {
	return s.repo.ListTodos(ctx, page, limit, cursor)
}
