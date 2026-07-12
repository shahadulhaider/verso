// Package logger provides structured JSON logging with OTel trace correlation.
package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

// New creates a JSON slog.Logger tagged with the given service name.
func New(serviceName string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(&traceHandler{
		inner:       handler,
		serviceName: serviceName,
	})
}

// WithContext stores a logger in the context.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext extracts the logger from the context, falling back to slog.Default.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// traceHandler wraps a slog.Handler to inject service_name, trace_id, and span_id.
type traceHandler struct {
	inner       slog.Handler
	serviceName string
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(slog.String("service_name", h.serviceName))

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.inner.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{
		inner:       h.inner.WithAttrs(attrs),
		serviceName: h.serviceName,
	}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{
		inner:       h.inner.WithGroup(name),
		serviceName: h.serviceName,
	}
}
