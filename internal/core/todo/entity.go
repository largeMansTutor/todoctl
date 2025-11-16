package todo

import "time"

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
