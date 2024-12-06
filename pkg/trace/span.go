package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("")

type Span interface {
	trace.Span
}

func Start(ctx context.Context, operation string, opts ...trace.SpanStartOption) (context.Context, Span) {
	return tracer.Start(ctx, operation)
}
