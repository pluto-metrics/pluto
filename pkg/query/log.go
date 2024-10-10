package query

import (
	"context"

	"go.uber.org/zap"
)

type contextKey string

var loggerContextKey contextKey = "queryLogger"

type typeLoggerContainer struct {
	logger *zap.Logger
}

func Log(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, &typeLoggerContainer{logger})
}

func LogWith(ctx context.Context, fields ...zap.Field) {
	l := ctx.Value(loggerContextKey)
	if l == nil {
		return
	}
	if c, ok := l.(*typeLoggerContainer); ok && c != nil && c.logger != nil {
		c.logger = c.logger.With(fields...)
	}
}

func loggerFromContext(ctx context.Context) *zap.Logger {
	l := ctx.Value(loggerContextKey)
	if l == nil {
		return nil
	}
	if c, ok := l.(*typeLoggerContainer); ok && c != nil {
		return c.logger
	}
	return nil
}
