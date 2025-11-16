package logger

import "go.uber.org/zap"

// New builds a production zap logger suitable for structured logging in the
// API server and background jobs.
func New() (*zap.Logger, error) {
	return zap.NewProduction()
}
