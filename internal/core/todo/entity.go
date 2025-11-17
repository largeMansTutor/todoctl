package todo

import (
	"fmt"
	"strings"
	"time"
)

// Todo represents a task item. Fields correspond to columns in the
// database. JSON tags define API serialization names.
//
// When adding new fields, update both the database migrations and
// service-level validation to ensure consistency.
type Todo struct {
	ID          uint64     `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description,omitempty" db:"description"`
	DueDate     *time.Time `json:"due_date,omitempty" db:"due_date"`
	Complete    bool       `json:"complete" db:"complete"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// TodoUpdate represents a partial update to a Todo. Fields that are nil
// will not be modified. The ID field is required to identify which
// record to update.
type TodoUpdate struct {
	ID          uint64      `json:"id"`
	Title       *string     `json:"title,omitempty"`
	Description **string    `json:"description,omitempty"`
	DueDate     **time.Time `json:"due_date,omitempty"`
	Complete    *bool       `json:"complete,omitempty"`
}

func (t TodoUpdate) Key() uint64 {
	return t.ID
}

func (t Todo) Key() string {
	return t.Title
}

func (t *Todo) Sanitize() error {
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(t.Title) > maxTitleLen {
		return fmt.Errorf("title too long (max %d)", maxTitleLen)
	}
	if t.Description != nil {
		d := strings.TrimSpace(*t.Description)
		if len(d) > maxDescLen {
			return fmt.Errorf("description too long (max %d)", maxDescLen)
		}
		if d == "" {
			t.Description = nil
		} else {
			t.Description = &d
		}
	}
	return nil
}

func (t *TodoUpdate) Sanitize() error {
	if t.Title != nil {
		title := strings.TrimSpace(*t.Title)
		if title == "" {
			return fmt.Errorf("title cannot be empty")
		}
		if len(title) > maxTitleLen {
			return fmt.Errorf("title too long (max %d)", maxTitleLen)
		}
		t.Title = &title
	}
	if t.Description != nil && *t.Description != nil {
		desc := strings.TrimSpace(**t.Description)
		if len(desc) > maxDescLen {
			return fmt.Errorf("description too long (max %d)", maxDescLen)
		}
		if desc == "" {
			t.Description = nil
		} else {
			t.Description = stringPtr(desc)
		}
	}
	return nil
}

const (
	maxTitleLen = 255
	maxDescLen  = 1024
)

func stringPtr(s string) **string {
	ptr := &s
	return &ptr
}
