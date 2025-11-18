// Package utils provides small helpers shared across components.
package utils

// Ptr returns a pointer to the provided value.
func Ptr[T any](v T) *T {
	return &v
}
