package mysqlstorage

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
)

// mysqlRepo implements the todo.Repository interface using a MySQL backend.
// It keeps SQL-specific details inside the adapter layer while exposing
// clean domain models to the rest of the application.
type mysqlRepo struct {
	db *sql.DB
}

// NewTodoRepository wires a MySQL-backed repository that satisfies the
// domain Repository interface.
func NewTodoRepository(db *sql.DB) tododomain.Repository {
	return &mysqlRepo{db: db}
}

// CreateTodos inserts the provided todo items into the database. It
// returns the created records as they exist in the database (with
// IDs and timestamps populated). If a title conflict occurs, it
// returns an error annotated with ErrDuplicateTitle.
func (m *mysqlRepo) CreateTodos(ctx context.Context, items []tododomain.Todo) ([]tododomain.Todo, error) {
	var created []tododomain.Todo
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO todos (title, description, due_date, complete) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for _, t := range items {
		res, err := stmt.ExecContext(ctx, t.Title, t.Description, t.DueDate, t.Complete)
		if err != nil {
			if isDuplicate(err) {
				return nil, fmt.Errorf("duplicate title: %w", tododomain.ErrDuplicateTitle)
			}
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		var todo tododomain.Todo
		row := tx.QueryRowContext(ctx, `SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos WHERE id = ?`, id)
		if err := row.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.DueDate, &todo.Complete, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			return nil, err
		}
		created = append(created, todo)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateTodos updates existing todos with the provided values. Only
// fields that are non-nil on the update struct are modified. It
// returns the updated records. Duplicate title conflicts propagate as
// ErrDuplicateTitle for the use case layer to translate.
func (m *mysqlRepo) UpdateTodos(ctx context.Context, updates []tododomain.TodoUpdate) ([]tododomain.Todo, error) {
	var out []tododomain.Todo
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	for _, u := range updates {
		setClauses := []string{}
		args := []interface{}{}
		if u.Title != nil {
			setClauses = append(setClauses, "title = ?")
			args = append(args, *u.Title)
		}
		if u.Description != nil {
			setClauses = append(setClauses, "description = ?")
			args = append(args, *u.Description)
		}
		if u.DueDate != nil {
			setClauses = append(setClauses, "due_date = ?")
			args = append(args, *u.DueDate)
		}
		if u.Complete != nil {
			setClauses = append(setClauses, "complete = ?")
			args = append(args, *u.Complete)
		}
		if len(setClauses) == 0 {
			continue
		}
		args = append(args, u.ID)
		query := fmt.Sprintf("UPDATE todos SET %s WHERE id = ?", joinClauses(setClauses))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			if isDuplicate(err) {
				return nil, fmt.Errorf("duplicate title: %w", tododomain.ErrDuplicateTitle)
			}
			return nil, err
		}
		var todo tododomain.Todo
		row := tx.QueryRowContext(ctx, `SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos WHERE id = ?`, u.ID)
		if err := row.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.DueDate, &todo.Complete, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		out = append(out, todo)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTodos implements both offset and keyset pagination. It returns the
// resulting todos sorted by id ascending and nextCursor if there are
// additional records.
func (m *mysqlRepo) ListTodos(ctx context.Context, page int, limit int, cursor string) ([]tododomain.Todo, string, error) {
	if limit <= 0 {
		return nil, "", fmt.Errorf("limit must be positive")
	}
	if limit > 100 {
		limit = 100
	}
	var args []interface{}
	var where string
	order := "ORDER BY id ASC"
	var nextCursor string
	if cursor != "" {
		idBytes, err := base64.URLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		id, err := strconv.ParseUint(string(idBytes), 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		where = "WHERE id > ?"
		args = append(args, id)
	} else {
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * limit
		args = append(args, limit, offset)
	}
	var query string
	if cursor != "" {
		query = fmt.Sprintf("SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos %s %s LIMIT ?", where, order)
		args = append(args, limit+1)
	} else {
		query = fmt.Sprintf("SELECT id, title, description, due_date, complete, created_at, updated_at FROM todos %s %s LIMIT ? OFFSET ?", where, order)
	}
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var todos []tododomain.Todo
	for rows.Next() {
		var todo tododomain.Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.DueDate, &todo.Complete, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			return nil, "", err
		}
		todos = append(todos, todo)
	}
	if cursor != "" && len(todos) > limit {
		last := todos[limit]
		nextCursor = base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", last.ID)))
		todos = todos[:limit]
	}
	return todos, nextCursor, nil
}

func joinClauses(clauses []string) string {
	result := ""
	for i, c := range clauses {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}

func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Error 1062")
}
