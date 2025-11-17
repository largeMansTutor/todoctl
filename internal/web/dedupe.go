package web

import (
	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
)

// DedupTodosUpdate and DedupTodos are aliases to the generic deduping slice types
// in the httpadapter package. They rely on the Key() implementations on the
// domain types to drop duplicates while unmarshalling.
type DedupTodosUpdate = httpadapter.DedupByKey[tododomain.TodoUpdate, uint64]
type DedupTodos = httpadapter.DedupByKey[tododomain.Todo, string]
