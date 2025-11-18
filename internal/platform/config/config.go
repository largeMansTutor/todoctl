package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/secrets"
)

// Config contains all configurable values for the application. It is loaded
// from environment variables via Viper. All fields are exported so they can
// be consumed by other packages and by the Fx dependency injection container.
//
// Security considerations:
//   - Secrets such as database credentials and API keys are provided via
//     environment variables rather than hard‑coded to avoid accidental
//     disclosure in source control.
//   - Timeouts and limits allow constraining resource usage, preventing
//     abuse through slowloris or large request bodies.
//   - Logging can be configured to avoid exposing sensitive data.
//
// Default values are sensible for local development; production deployments
// should override them via environment variables.
type Config struct {
	httpadapter.HTTPConfig `mapstructure:",squash"`
	// Database connection string (DSN). The DSN should be in the
	// format accepted by go-sql-driver/mysql. It can include
	// connection parameters (e.g. charset, parseTime, loc).
	DatabaseDSN string `mapstructure:"DB_DSN"`
	// OpenAPIPath points to the OpenAPI spec file to serve.
	OpenAPIPath string `mapstructure:"APP_OPENAPI_PATH"`

	// APIKey, if set, enables API key authentication. Requests must
	// include the X-API-Key header with this value to be accepted. If
	// unset (empty), the middleware is a no-op and all requests are
	// allowed. Keeping the API key outside source code prevents
	// accidental disclosure.
	APIKey string `mapstructure:"APP_API_KEY"`
	// AllowedAPIKeys optionally configures multiple API keys with labels in
	// the form "key:label,key2:label2". If empty, falls back to APIKey. Labels
	// are used only for logging/audit.
	AllowedAPIKeys string `mapstructure:"APP_ALLOWED_API_KEYS"`
	// AllowedAPIKeysHashed configures bcrypt-hashed API keys keyed by id in the
	// form "id1:$2b$...[:label],id2:$2b$...". Caller must send X-API-Key-ID=id
	// and X-API-Key with the cleartext secret. Labels are for logging/audit.
	AllowedAPIKeysHashed string `mapstructure:"APP_ALLOWED_API_KEYS_HASHED"`

	// SecretsProvider selects an external secrets manager ("vault" supported). If empty,
	// values come from env directly.
	SecretsProvider string `mapstructure:"APP_SECRETS_PROVIDER"`
	// Vault settings (used when SecretsProvider=vault).
	VaultAddr  string `mapstructure:"VAULT_ADDR"`
	VaultToken string `mapstructure:"VAULT_TOKEN"`
	VaultMount string `mapstructure:"VAULT_MOUNT"`
	VaultPath  string `mapstructure:"VAULT_KV_PATH"`

	// Rate limiting configuration (per-IP or per-API-key).
	RateLimitPerSecond int `mapstructure:"APP_RATE_LIMIT_PER_SECOND"`
	RateLimitBurst     int `mapstructure:"APP_RATE_LIMIT_BURST"`
}

// New loads configuration values from environment variables. If a value
// is not provided it falls back to the default defined here. Viper allows
// reading from a .env file or the environment; here we rely on env.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("APP_PORT", 8080)
	v.SetDefault("APP_READ_TIMEOUT", 10*time.Second)
	v.SetDefault("APP_WRITE_TIMEOUT", 10*time.Second)
	v.SetDefault("APP_IDLE_TIMEOUT", 60*time.Second)
	v.SetDefault("APP_MAX_BODY_BYTES", 1<<20) // 1 MiB
	v.SetDefault("APP_TLS_CERT_FILE", "")
	v.SetDefault("APP_TLS_KEY_FILE", "")

	v.SetDefault("DB_DSN", "root:password@tcp(localhost:3306)/todos?parseTime=true&charset=utf8mb4&loc=UTC")
	v.SetDefault("APP_API_KEY", "")
	v.SetDefault("APP_ALLOWED_API_KEYS", "")
	v.SetDefault("APP_ALLOWED_API_KEYS_HASHED", "")
	v.SetDefault("APP_SECRETS_PROVIDER", "")
	v.SetDefault("VAULT_ADDR", "")
	v.SetDefault("VAULT_TOKEN", "")
	v.SetDefault("VAULT_MOUNT", "secret")
	v.SetDefault("VAULT_KV_PATH", "")
	v.SetDefault("APP_RATE_LIMIT_PER_SECOND", 20)
	v.SetDefault("APP_RATE_LIMIT_BURST", 40)
	v.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	v.SetDefault("OTEL_SERVICE_NAME", "todo-api")
	v.SetDefault("APP_OPENAPI_PATH", "openapi/openapi.yaml")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to load configuration: %w", err)
	}
	applySecrets(&cfg)
	return &cfg, nil
}

// applySecrets optionally overrides config values from an external secrets backend.
func applySecrets(cfg *Config) {
	switch cfg.SecretsProvider {
	case "vault":
		if cfg.VaultAddr == "" || cfg.VaultToken == "" || cfg.VaultPath == "" {
			return
		}
		client, err := secrets.NewVaultProvider(cfg.VaultAddr, cfg.VaultToken, cfg.VaultMount, cfg.VaultPath)
		if err != nil {
			return
		}
		secretMap, err := client.Fetch()
		if err != nil {
			return
		}
		override := func(key string, target *string) {
			if target == nil {
				return
			}
			if val, ok := secretMap[key]; ok && val != "" {
				*target = val
			}
		}
		override("DB_DSN", &cfg.DatabaseDSN)
		override("APP_API_KEY", &cfg.APIKey)
		override("APP_ALLOWED_API_KEYS", &cfg.AllowedAPIKeys)
		override("APP_ALLOWED_API_KEYS_HASHED", &cfg.AllowedAPIKeysHashed)
	default:
	}
}
