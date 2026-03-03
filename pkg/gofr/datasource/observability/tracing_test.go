package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestAddTrace_WithNilTracer(t *testing.T) {
	instrumenter := NewInstrumentation("mongo")

	ctx := context.Background()
	query := newMockQuery("insertOne")

	newCtx, span := instrumenter.AddTrace(ctx, query)

	// Should return original context and nil span
	assert.Equal(t, ctx, newCtx)
	assert.Nil(t, span)
}

func TestAddTrace_WithTracer(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	instrumenter := NewInstrumentation("mongo")
	instrumenter.SetTracer(tp.Tracer("test"))

	ctx := context.Background()
	query := newMockQuery("insertOne")

	newCtx, span := instrumenter.AddTrace(ctx, query)

	require.NotNil(t, span)
	assert.NotEqual(t, ctx, newCtx)
	assert.True(t, span.SpanContext().IsValid())

	span.End()

	// Verify span name
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "mongo-insertOne", spans[0].Name)
}

func TestAddTrace_SetsAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	instrumenter := NewInstrumentation("arango")
	instrumenter.SetTracer(tp.Tracer("test"))

	ctx := context.Background()
	query := newMockQuery("query")
	query.traceLabels = map[string]string{
		"collection": "users",
		"database":   "testdb",
	}

	_, span := instrumenter.AddTrace(ctx, query)
	require.NotNil(t, span)
	span.End()

	// Verify attributes are prefixed with datasource name
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	attrMap := make(map[string]string)

	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value.AsString()
	}

	assert.Equal(t, "users", attrMap["arango.collection"])
	assert.Equal(t, "testdb", attrMap["arango.database"])
}

func TestSpanName(t *testing.T) {
	tests := []struct {
		name           string
		datasourceName string
		operation      string
		expected       string
	}{
		{
			name:           "mongo insertOne",
			datasourceName: "mongo",
			operation:      "insertOne",
			expected:       "mongo-insertOne",
		},
		{
			name:           "redis get",
			datasourceName: "redis",
			operation:      "get",
			expected:       "redis-get",
		},
		{
			name:           "arango query",
			datasourceName: "arangodb",
			operation:      "query",
			expected:       "arangodb-query",
		},
		{
			name:           "empty operation",
			datasourceName: "test",
			operation:      "",
			expected:       "test-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &instrumentation{datasourceName: tt.datasourceName}
			result := inst.spanName(tt.operation)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSpanDurationKey(t *testing.T) {
	tests := []struct {
		name           string
		datasourceName string
		method         string
		expected       string
	}{
		{
			name:           "mongo insert",
			datasourceName: "mongo",
			method:         "insert",
			expected:       "mongo.insert.duration",
		},
		{
			name:           "redis set",
			datasourceName: "redis",
			method:         "set",
			expected:       "redis.set.duration",
		},
		{
			name:           "arangodb query",
			datasourceName: "arangodb",
			method:         "query",
			expected:       "arangodb.query.duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &instrumentation{datasourceName: tt.datasourceName}
			result := inst.spanDurationKey(tt.method)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTraceLabelsForDB(t *testing.T) {
	inst := &instrumentation{datasourceName: "mongo"}

	tests := []struct {
		name     string
		labels   map[string]string
		expected map[string]string
	}{
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "single label",
			labels: map[string]string{
				"collection": "users",
			},
			expected: map[string]string{
				"mongo.collection": "users",
			},
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"collection": "users",
				"database":   "testdb",
				"host":       "localhost",
			},
			expected: map[string]string{
				"mongo.collection": "users",
				"mongo.database":   "testdb",
				"mongo.host":       "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inst.getTraceLabelsForDB(tt.labels)

			assert.Len(t, result, len(tt.expected))

			resultMap := make(map[string]string)
			for _, kv := range result {
				resultMap[string(kv.Key)] = kv.Value.AsString()
			}

			for key, value := range tt.expected {
				assert.Equal(t, value, resultMap[key])
			}
		})
	}
}

func TestGetTraceLabelsForDB_DifferentDatasources(t *testing.T) {
	tests := []struct {
		name           string
		datasourceName string
		labels         map[string]string
		expectedPrefix string
	}{
		{
			name:           "redis datasource",
			datasourceName: "redis",
			labels:         map[string]string{"key": "mykey"},
			expectedPrefix: "redis.",
		},
		{
			name:           "arangodb datasource",
			datasourceName: "arangodb",
			labels:         map[string]string{"collection": "users"},
			expectedPrefix: "arangodb.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &instrumentation{datasourceName: tt.datasourceName}
			result := inst.getTraceLabelsForDB(tt.labels)

			for _, kv := range result {
				assert.Contains(t, string(kv.Key), tt.expectedPrefix)
			}
		})
	}
}
