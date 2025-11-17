package mysqlstorage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
)

func TestBuildUpdateQuery(t *testing.T) {
	title := "new"
	desc := "desc"
	when := time.Now()
	done := true

	query, clauses, args, ok := buildUpdateQuery(tododomain.TodoUpdate{
		ID:          1,
		Title:       &title,
		Description: stringPtr(desc),
		DueDate:     timePtr(when),
		Complete:    &done,
	}, []string{}, []any{})

	require.True(t, ok)
	require.Equal(t, []string{"title = ?", "description = ?", "due_date = ?", "complete = ?"}, clauses)
	require.Equal(t, []any{title, &desc, &when, true, uint64(1)}, args)
	require.Equal(t, "UPDATE todos SET title = ?, description = ?, due_date = ?, complete = ? WHERE id = ?", query)
}

func stringPtr(s string) **string {
	ptr := &s
	return &ptr
}

func timePtr(t time.Time) **time.Time {
	ptr := &t
	return &ptr
}

func ptr[T any](v T) *T {
	return &v
}
