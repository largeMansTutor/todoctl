package todo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTodoSanitize(t *testing.T) {
	desc := "  desc  "
	todo := Todo{
		Title:       "  title  ",
		Description: &desc,
	}
	err := todo.Sanitize()
	require.NoError(t, err)
	require.Equal(t, "title", todo.Title)
	require.Equal(t, "desc", *todo.Description)
}

func TestTodoSanitizeInvalid(t *testing.T) {
	todo := Todo{Title: "   "}
	err := todo.Sanitize()
	require.Error(t, err)
}

func TestTodoUpdateSanitize(t *testing.T) {
	title := "  updated "
	desc := "  desc  "
	update := TodoUpdate{
		ID:          1,
		Title:       &title,
		Description: stringPtr(desc),
	}
	err := update.Sanitize()
	require.NoError(t, err)
	require.Equal(t, "updated", *update.Title)
	require.Equal(t, "desc", **update.Description)
}

func TestTodoUpdateSanitizeEmptyTitle(t *testing.T) {
	title := "   "
	update := TodoUpdate{ID: 1, Title: &title}
	err := update.Sanitize()
	require.Error(t, err)
}
