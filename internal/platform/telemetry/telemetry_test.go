package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
)

func TestConfigureTracerProvider_NoEndpoint(t *testing.T) {
	cfg := &config.Config{
		HTTPConfig: httpadapter.HTTPConfig{
			OTELConfig: httpadapter.OTELConfig{
				OTELServiceName:      "test",
				OTELExporterEndpoint: "",
			},
		},
	}
	logger := zap.NewNop()
	app := fx.New(
		fx.Provide(func() *config.Config { return cfg }),
		fx.Provide(func() *zap.Logger { return logger }),
		fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) error {
			_, err := ConfigureTracerProvider(lc, cfg, logger)
			require.NoError(t, err)
			return nil
		}),
	)
	require.NoError(t, app.Start(context.Background()))
	require.NoError(t, app.Stop(context.Background()))
}
