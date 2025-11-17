package assets

import "embed"

// FS contains embedded project assets (migrations, OpenAPI spec) so the CLI
// works after `go install` without needing the source tree present.
//
//go:embed migrations/*.sql openapi/*
var FS embed.FS

const (
	MigrationsDir = "migrations"
	OpenAPIFile   = "openapi/openapi.yaml"
)
