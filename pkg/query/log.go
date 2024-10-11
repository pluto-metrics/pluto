package query

import (
	"context"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type contextKey string

var loggerContextKey contextKey = "queryLogger"

type typeLoggerContainer struct {
	sync.Mutex
	logger *zap.Logger
}

func Log(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, &typeLoggerContainer{logger: logger})
}

func LogWith(ctx context.Context, fields ...zap.Field) {
	l := ctx.Value(loggerContextKey)
	if l == nil {
		return
	}
	if c, ok := l.(*typeLoggerContainer); ok && c != nil && c.logger != nil {
		c.Lock()
		c.logger = c.logger.With(fields...)
		c.Unlock()
	}
}

func loggerFromContext(ctx context.Context) *zap.Logger {
	l := ctx.Value(loggerContextKey)
	if l == nil {
		return nil
	}
	if c, ok := l.(*typeLoggerContainer); ok && c != nil {
		c.Lock()
		defer c.Unlock()
		return c.logger
	}
	return nil
}

func logQueryFinished(ctx context.Context, err error) {
	l := loggerFromContext(ctx)
	if l == nil {
		return
	}
	if err != nil {
		l.Error("query failed", zap.Error(err))
	} else {
		l.Debug("query success")
	}
}

func Format(q string) string {
	a := strings.Split(q, "\n")
	for i := 0; i < len(a); i++ {
		a[i] = strings.TrimSpace(a[i])
	}
	return strings.TrimSpace(strings.Join(a, " "))
}
