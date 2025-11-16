package idempotency

import (
	"context"
	"time"
)

// Record captures the stored response for an idempotent call. It is used by
// the use cases to short-circuit repeated requests without re-running
// business logic.
type Record struct {
	Key        string
	Method     string
	Path       string
	StatusCode int
	Response   []byte
	CreatedAt  time.Time
}

// Store describes the persistence boundary for idempotency data. Different
// backends (SQL, Redis, etc.) can implement this interface without
// impacting business logic.
type Store interface {
	Get(ctx context.Context, key, scope string) (*Record, error)
	Save(ctx context.Context, key, scope string, status int, response []byte) error
}
