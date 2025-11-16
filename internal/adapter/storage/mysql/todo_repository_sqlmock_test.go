package mysqlstorage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
)

func TestCreateTodos_WithSQLMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	repo := NewTodoRepository(db)
	items := []tododomain.Todo{{Title: "task", Complete: false}}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO todos").ExpectExec().
		WithArgs("task", nil, nil, false).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos WHERE id = ?").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "due_date", "complete", "created_at", "updated_at"}).
			AddRow(uint64(1), "task", nil, time.Now(), false, time.Now(), time.Now()))
	mock.ExpectCommit()

	created, err := repo.CreateTodos(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateTodos_WithSQLMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	repo := NewTodoRepository(db)
	title := "new-title"
	complete := true
	updates := []tododomain.TodoUpdate{{ID: 1, Title: &title, Complete: &complete}}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE todos SET title = ?, complete = ? WHERE id = ?").
		WithArgs(title, complete, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos WHERE id = ?").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "due_date", "complete", "created_at", "updated_at"}).
			AddRow(uint64(1), title, nil, sql.NullTime{}, complete, time.Now(), time.Now()))
	mock.ExpectCommit()

	updated, err := repo.UpdateTodos(context.Background(), updates)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListTodos_WithSQLMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	repo := NewTodoRepository(db)
	mock.ExpectQuery("SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos .*LIMIT ? OFFSET ?").
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "due_date", "complete", "created_at", "updated_at"}).
			AddRow(uint64(1), "title", nil, sql.NullTime{}, false, time.Now(), time.Now()))

	todos, next, err := repo.ListTodos(context.Background(), 1, 10, "")
	require.NoError(t, err)
	require.Len(t, todos, 1)
	require.Equal(t, "", next)
	require.NoError(t, mock.ExpectationsWereMet())
}
