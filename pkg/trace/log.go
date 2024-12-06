package trace

import (
	"context"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log returns global logger
func Log(ctx context.Context) otelzap.LoggerWithCtx {
	return otelzap.Ctx(ctx)
}

func Error(err error) zap.Field {
	// you could even flip the encoding depending on the environment here if you want
	return zap.Field{Key: "error", Type: zapcore.StringType, String: err.Error()}
}
