package scope

import (
	"context"

	"go.uber.org/zap"
)

func With(ctx context.Context, fields ...zap.Field) context.Context {
	return WithLogger(ctx, Logger(ctx).With(fields...))
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKeyZapLogger, logger)
}

func Logger(ctx context.Context) *zap.Logger {
	logger := contextGet[*zap.Logger](ctx, contextKeyZapLogger)
	if logger == nil {
		logger = zap.L()
	}
	return logger
}
