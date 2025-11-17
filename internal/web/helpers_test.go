package web

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
)

func TestDedupTodos(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectTitles []string
		expectErr    bool
	}{
		{
			name:         "dedupes by title keeps first occurrence",
			input:        `[{"title":"a"},{"title":"a"},{"title":"b"}]`,
			expectTitles: []string{"a", "b"},
		},
		{
			name:      "invalid json fails",
			input:     `{"not":"array"}`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var todos DedupTodos
			err := json.Unmarshal([]byte(tt.input), &todos)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, todos, len(tt.expectTitles))
			for i, title := range tt.expectTitles {
				assert.Equal(t, title, todos[i].Title)
			}
		})
	}
}

func TestDedupTodosUpdate(t *testing.T) {
	payload := `[{"id":1,"title":"x"},{"id":1,"title":"y"},{"id":2,"title":"z"}]`
	var updates DedupTodosUpdate

	err := json.Unmarshal([]byte(payload), &updates)
	require.NoError(t, err)

	require.Len(t, updates, 2)
	assert.Equal(t, uint64(1), updates[0].ID)
	assert.Equal(t, "x", derefString(updates[0].Title))
	assert.Equal(t, uint64(2), updates[1].ID)
	assert.Equal(t, "z", derefString(updates[1].Title))
}

func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Ensure TodoUpdate implements the key interface at compile time.
var _ httpadapter.Keyer[uint64] = tododomain.TodoUpdate{}
