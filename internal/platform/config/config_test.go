package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewDefaults(t *testing.T) {
	// Ensure env is clean for defaults
	os.Clearenv()
	cfg, err := New()
	require.NoError(t, err)
	require.Equal(t, 8080, cfg.HTTPPort)
	require.Equal(t, 10*time.Second, cfg.ReadTimeout)
	require.Equal(t, "todo-api", cfg.OTELServiceName)
	require.Equal(t, "root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC", cfg.DatabaseDSN)
	require.Equal(t, "openapi/openapi.yaml", cfg.OpenAPIPath)
	require.Equal(t, "", cfg.AllowedAPIKeys)
	require.Equal(t, "", cfg.AllowedAPIKeysHashed)
}
