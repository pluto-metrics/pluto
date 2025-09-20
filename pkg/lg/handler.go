package lg

import (
	"context"
	"log/slog"
)

type ctxKey string

const (
	slogFields ctxKey = "slogAttrs"
)

// Handler ...
type Handler struct {
	slog.Handler
}

// NewHandler ...
func NewHandler(h slog.Handler) slog.Handler {
	return Handler{Handler: h}
}

// Handle adds contextual attributes to the Record before calling the underlying
// handler
func (h Handler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.Handler.Handle(ctx, r)
}

// With adds an slog attribute to the provided context so that it will be
// included in any Record created with such context
func With(parent context.Context, attr ...slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(slogFields).([]slog.Attr); ok {
		v = append(v, attr...)
		return context.WithValue(parent, slogFields, v)
	}

	v := []slog.Attr{}
	v = append(v, attr...)
	return context.WithValue(parent, slogFields, v)
}
