package mysqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/idempotency"
)

type idempotencyStore struct {
	db *sql.DB
}

const storeMethod = "USECASE"

// NewIdempotencyStore returns a Store backed by MySQL. The key is hashed using
// xxhash which is fast and collision-free.
func NewIdempotencyStore(db *sql.DB) idempotency.Store {
	return &idempotencyStore{db: db}
}

func (s *idempotencyStore) Get(ctx context.Context, key, scope string) (*idempotency.Record, error) {
	hkey := hashKey(key)
	row := s.db.QueryRowContext(ctx,
		`SELECT method, path, status_code, response_body, created_at
         FROM idempotency_keys WHERE key_hash = ? AND method = ? AND path = ?`,
		hkey, storeMethod, scope,
	)
	var rec idempotency.Record
	var body []byte
	if err := row.Scan(&rec.Method, &rec.Path, &rec.StatusCode, &body, &rec.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rec.Key = key
	rec.Response = body
	return &rec, nil
}

func (s *idempotencyStore) Save(ctx context.Context, key, path string, status int, response []byte) error {
	hkey := hashKey(key)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO idempotency_keys (key_hash, method, path, status_code, response_body, created_at)
         VALUES (?, ?, ?, ?, ?, NOW())
         ON DUPLICATE KEY UPDATE
           status_code = VALUES(status_code),
           response_body = VALUES(response_body),
           created_at = VALUES(created_at)`,
		hkey, storeMethod, path, status, response,
	)
	return err
}

func hashKey(key string) string {
	sum := xxhash.Sum64String(key)
	return fmt.Sprintf("%016x", sum)
}
