package otel_test

import (
	"context"
	"testing"

	versotel "github.com/shahadulhaider/verso/libs/go/otel"
	"go.opentelemetry.io/otel"
)

func TestInit_NoExporter(t *testing.T) {
	// Set a bogus endpoint so Init() will still construct the provider
	// without connecting to a real collector. The OTLP exporter uses
	// async batching, so Init itself won't fail on a bad endpoint.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:0")

	ctx := context.Background()
	shutdown, err := versotel.Init(ctx, "test-service")
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown func should not be nil")
	}

	// Verify a global tracer provider was set (not the noop one)
	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Fatal("tracer provider should be set")
	}

	// Shutdown cleanly
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
