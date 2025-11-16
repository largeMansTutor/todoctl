package mysqlstorage

import "testing"

func TestJoinClauses(t *testing.T) {
	clauses := []string{"title = ?", "description = ?", "complete = ?"}
	result := joinClauses(clauses)
	expected := "title = ?, description = ?, complete = ?"
	if result != expected {
		t.Fatalf("unexpected join result: %s", result)
	}
}
