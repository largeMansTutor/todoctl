package httpadapter

import "time"

type HTTPConfig struct {
	OTELConfig `mapstructure:",squash"`
	// HTTP server configuration
	HTTPPort     int           `mapstructure:"APP_PORT"`
	ReadTimeout  time.Duration `mapstructure:"APP_READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"APP_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `mapstructure:"APP_IDLE_TIMEOUT"`
	MaxBodyBytes int64         `mapstructure:"APP_MAX_BODY_BYTES"`
	TLSCertFile  string        `mapstructure:"APP_TLS_CERT_FILE"`
	TLSKeyFile   string        `mapstructure:"APP_TLS_KEY_FILE"`
	OpenAPIPath  string        `mapstructure:"APP_OPENAPI_PATH"`
}

type OTELConfig struct {
	OTELExporterEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELServiceName      string `mapstructure:"OTEL_SERVICE_NAME"`
}
