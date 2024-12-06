package prom

import (
	"context"

	"github.com/pluto-metrics/pluto/pkg/trace"
	"go.uber.org/zap"
)

type errorLevel interface {
	String() string
}

type logger struct {
}

func (l *logger) Log(keyvals ...interface{}) error {
	ctx := context.Background()
	var msg string
	var level errorLevel
	fields := make([]zap.Field, 0)

	for i := 1; i < len(keyvals); i += 2 {
		keyObj := keyvals[i-1]
		keyStr, ok := keyObj.(string)
		if !ok {
			trace.Log(ctx).Error("can't handle log, wrong key", zap.Any("keyvals", keyvals))
			return nil
		}

		if keyStr == "level" {
			level, ok = keyvals[i].(errorLevel)
			if !ok {
				trace.Log(ctx).Error("can't handle log, wrong level", zap.Any("keyvals", keyvals))
				return nil
			}
			continue
		}

		if keyStr == "msg" {
			msg, ok = keyvals[i].(string)
			if !ok {
				trace.Log(ctx).Error("can't handle log, wrong msg", zap.Any("keyvals", keyvals))
				return nil
			}
			continue
		}

		fields = append(fields, zap.Any(keyStr, keyvals[i]))
	}

	switch level.String() {
	case "debug":
		trace.Log(ctx).Debug(msg, fields...)
	case "info":
		trace.Log(ctx).Info(msg, fields...)
	case "warn":
		trace.Log(ctx).Warn(msg, fields...)
	case "error":
		trace.Log(ctx).Error(msg, fields...)
	default:
		trace.Log(ctx).Error("can't handle log, unknown level", fields...)
		return nil
	}
	return nil
}
