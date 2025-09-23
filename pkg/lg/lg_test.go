package lg

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestError(t *testing.T) {
	err := errors.New("test error")
	attr := Error(err)
	if attr.Key != "error" {
		t.Errorf("Error attr key = %s; want 'error'", attr.Key)
	}
	if attr.Value.String() != "test error" {
		t.Errorf("Error attr value = %s; want 'test error'", attr.Value.String())
	}
}

func TestHandlerHandle(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{})
	wrapped := NewHandler(h)

	ctx := With(context.Background(), slog.String("key", "value"))
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)

	err := wrapped.Handle(ctx, r)
	if err != nil {
		t.Errorf("Handle returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "key=value") {
		t.Errorf("Output does not contain 'key=value': %s", output)
	}
}

func TestHandlerHandleWithoutContext(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{})
	wrapped := NewHandler(h)

	ctx := context.Background()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)

	err := wrapped.Handle(ctx, r)
	if err != nil {
		t.Errorf("Handle returned error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "key=value") {
		t.Errorf("Output should not contain 'key=value': %s", output)
	}
}

func TestWith(t *testing.T) {
	ctx := With(context.Background(), slog.String("key1", "value1"))
	ctx = With(ctx, slog.String("key2", "value2"))

	attrs, ok := ctx.Value(slogFields).([]slog.Attr)
	if !ok {
		t.Fatal("Context should contain slogAttrs")
	}
	if len(attrs) != 2 {
		t.Errorf("Expected 2 attrs, got %d", len(attrs))
	}
	if attrs[0].Key != "key1" || attrs[0].Value.String() != "value1" {
		t.Errorf("First attr incorrect: %v", attrs[0])
	}
	if attrs[1].Key != "key2" || attrs[1].Value.String() != "value2" {
		t.Errorf("Second attr incorrect: %v", attrs[1])
	}
}

func TestWithNilContext(t *testing.T) {
	ctx := With(nil, slog.String("key", "value"))
	if ctx == nil {
		t.Fatal("With should not return nil")
	}
	attrs, ok := ctx.Value(slogFields).([]slog.Attr)
	if !ok {
		t.Fatal("Context should contain slogAttrs")
	}
	if len(attrs) != 1 || attrs[0].Key != "key" {
		t.Errorf("Attr not set correctly: %v", attrs)
	}
}
