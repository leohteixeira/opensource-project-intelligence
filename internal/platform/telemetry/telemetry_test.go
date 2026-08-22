package telemetry_test

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// IT-134: trace context survives every durable boundary without sensitive payloads.
func TestIT134TraceContextSurvivesEveryDurableBoundaryWithoutPayloads(t *testing.T) {
	t.Parallel()
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled},
	))
	for _, boundary := range []string{"api", "outbox", "jetstream", "worker", "sql", "s3", "model-tool"} {
		carrier := propagation.MapCarrier{}
		propagator.Inject(ctx, carrier)
		for key, value := range carrier {
			serialized := strings.ToLower(key + ":" + value)
			for _, forbidden := range []string{"authorization", "cookie", "password", "token", "payload"} {
				if strings.Contains(serialized, forbidden) {
					t.Fatalf("%s boundary propagated forbidden content %q", boundary, serialized)
				}
			}
		}
		ctx = propagator.Extract(context.Background(), carrier)
		if got := trace.SpanContextFromContext(ctx); !got.IsValid() || got.TraceID() != traceID {
			t.Fatalf("%s boundary trace context = %s, want %s", boundary, got.TraceID(), traceID)
		}
	}
}
