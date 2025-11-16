package mysqlstorage

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyStore_GetAndSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewIdempotencyStore(db)
	hkey := hashKey("abc")

	mock.ExpectQuery("SELECT method, path, status_code, response_body, created_at").
		WithArgs(hkey, storeMethod, "todos:create").
		WillReturnRows(sqlmock.NewRows([]string{"method", "path", "status_code", "response_body", "created_at"}).
			AddRow(storeMethod, "todos:create", 201, []byte(`{"todos":[]}`), time.Now()))

	rec, err := store.Get(context.Background(), "abc", "todos:create")
	require.NoError(t, err)
	require.NotNil(t, rec)
	require.Equal(t, "abc", rec.Key)

	mock.ExpectExec("INSERT INTO idempotency_keys").
		WithArgs(hkey, storeMethod, "todos:create", 201, []byte("body")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, store.Save(context.Background(), "abc", "todos:create", 201, []byte("body")))
	require.NoError(t, mock.ExpectationsWereMet())
}
