package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
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
	// HTTP server configuration
	HTTPPort     int           `mapstructure:"APP_PORT"`
	ReadTimeout  time.Duration `mapstructure:"APP_READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"APP_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `mapstructure:"APP_IDLE_TIMEOUT"`
	MaxBodyBytes int64         `mapstructure:"APP_MAX_BODY_BYTES"`
	TLSCertFile  string        `mapstructure:"APP_TLS_CERT_FILE"`
	TLSKeyFile   string        `mapstructure:"APP_TLS_KEY_FILE"`

	// Database connection string (DSN). The DSN should be in the
	// format accepted by go-sql-driver/mysql. It can include
	// connection parameters (e.g. charset, parseTime, loc).
	DatabaseDSN string `mapstructure:"DB_DSN"`

	// APIKey, if set, enables API key authentication. Requests must
	// include the X-API-Key header with this value to be accepted. If
	// unset (empty), the middleware is a no-op and all requests are
	// allowed. Keeping the API key outside source code prevents
	// accidental disclosure.
	APIKey string `mapstructure:"APP_API_KEY"`

	// Rate limiting configuration (per-IP or per-API-key).
	RateLimitPerSecond int `mapstructure:"APP_RATE_LIMIT_PER_SECOND"`
	RateLimitBurst     int `mapstructure:"APP_RATE_LIMIT_BURST"`

	// Observability
	OTELExporterEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELServiceName      string `mapstructure:"OTEL_SERVICE_NAME"`
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
	v.SetDefault("APP_RATE_LIMIT_PER_SECOND", 20)
	v.SetDefault("APP_RATE_LIMIT_BURST", 40)
	v.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	v.SetDefault("OTEL_SERVICE_NAME", "todo-api")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to load configuration: %w", err)
	}
	return &cfg, nil
}
