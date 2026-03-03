package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewInstrumentation(t *testing.T) {
	tests := []struct {
		name           string
		datasourceName string
	}{
		{"mongo datasource", "mongo"},
		{"arangodb datasource", "arangodb"},
		{"redis datasource", "redis"},
		{"empty name", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrumenter := NewInstrumentation(tt.datasourceName)

			assert.NotNil(t, instrumenter)
		})
	}
}

func TestNewInstrumentation_HasNoopDefaults(t *testing.T) {
	instrumenter := NewInstrumentation("test")

	// Logging and metrics should not panic without explicit setup
	assert.NotPanics(t, func() {
		instrumenter.Debugf("debug message %s", "value")
	})
	assert.NotPanics(t, func() {
		instrumenter.Logf("log message %s", "value")
	})
	assert.NotPanics(t, func() {
		instrumenter.Errorf("error message %s", "value")
	})
	assert.NotPanics(t, func() {
		instrumenter.RegisterStatsHistogram(0.1, 0.5, 1.0)
	})
}

func TestSetLogger_WithNil(t *testing.T) {
	instrumenter := NewInstrumentation("test")

	// Set a valid logger first
	mockLog := newMockLogger()
	instrumenter.SetLogger(mockLog)

	// Setting nil should not replace the existing logger
	instrumenter.SetLogger(nil)

	// Should still work with the original logger
	instrumenter.Debugf("test message")

	assert.Len(t, mockLog.debugfCalls, 1)
}

func TestSetLogger_WithValidLogger(t *testing.T) {
	instrumenter := NewInstrumentation("test")

	mockLog := newMockLogger()
	instrumenter.SetLogger(mockLog)

	// Verify delegation works
	instrumenter.Debugf("debug %s", "value")
	instrumenter.Logf("log %s", "value")
	instrumenter.Errorf("error %s", "value")

	assert.Len(t, mockLog.debugfCalls, 1)
	assert.Equal(t, "debug %s", mockLog.debugfCalls[0].format)

	assert.Len(t, mockLog.logfCalls, 1)
	assert.Equal(t, "log %s", mockLog.logfCalls[0].format)

	assert.Len(t, mockLog.errorfCalls, 1)
	assert.Equal(t, "error %s", mockLog.errorfCalls[0].format)
}

func TestSetMetrics_WithNil(t *testing.T) {
	instrumenter := NewInstrumentation("test")

	// Set a valid metrics first
	mockMet := newMockMetrics()
	instrumenter.SetMetrics(mockMet)

	// Setting nil should not replace the existing metrics
	instrumenter.SetMetrics(nil)

	// Should still work with the original metrics
	instrumenter.RegisterStatsHistogram(0.1, 0.5)

	assert.Len(t, mockMet.newHistogramCalls, 1)
}

func TestSetMetrics_WithValidMetrics(t *testing.T) {
	instrumenter := NewInstrumentation("test")

	mockMet := newMockMetrics()
	instrumenter.SetMetrics(mockMet)

	instrumenter.RegisterStatsHistogram(0.1, 0.5, 1.0)

	require.Len(t, mockMet.newHistogramCalls, 1)
	assert.Equal(t, "app_test_stats", mockMet.newHistogramCalls[0].name)
	assert.Equal(t, []float64{0.1, 0.5, 1.0}, mockMet.newHistogramCalls[0].buckets)
}

func TestSetTracer_WithNil(t *testing.T) {
	instrumenter := NewInstrumentation("test")

	// Setting nil tracer should not panic
	assert.NotPanics(t, func() {
		instrumenter.SetTracer(nil)
	})

	// AddTrace should handle nil tracer gracefully
	ctx := context.Background()
	query := newMockQuery("operation")

	newCtx, span := instrumenter.AddTrace(ctx, query)

	assert.Equal(t, ctx, newCtx)
	assert.Nil(t, span)
}

func TestSetTracer_WithValidTracer(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	instrumenter := NewInstrumentation("test")
	instrumenter.SetTracer(tp.Tracer("test-tracer"))

	ctx := context.Background()
	query := newMockQuery("operation")

	newCtx, span := instrumenter.AddTrace(ctx, query)

	assert.NotEqual(t, ctx, newCtx)
	require.NotNil(t, span)
	assert.True(t, span.SpanContext().IsValid())

	span.End()
}

func TestLoggingMethods_Delegation(t *testing.T) {
	instrumenter := NewInstrumentation("test")
	mockLog := newMockLogger()
	instrumenter.SetLogger(mockLog)

	// Test Debugf
	instrumenter.Debugf("debug format %d", 42)
	require.Len(t, mockLog.debugfCalls, 1)
	assert.Equal(t, "debug format %d", mockLog.debugfCalls[0].format)
	assert.Equal(t, []any{42}, mockLog.debugfCalls[0].args)

	// Test Logf
	instrumenter.Logf("log format %s", "test")
	require.Len(t, mockLog.logfCalls, 1)
	assert.Equal(t, "log format %s", mockLog.logfCalls[0].format)
	assert.Equal(t, []any{"test"}, mockLog.logfCalls[0].args)

	// Test Errorf
	instrumenter.Errorf("error format %v", "error")
	require.Len(t, mockLog.errorfCalls, 1)
	assert.Equal(t, "error format %v", mockLog.errorfCalls[0].format)
	assert.Equal(t, []any{"error"}, mockLog.errorfCalls[0].args)
}

func TestRegisterStatsHistogram(t *testing.T) {
	tests := []struct {
		name           string
		datasourceName string
		expectedName   string
		expectedDesc   string
	}{
		{
			name:           "mongo datasource",
			datasourceName: "mongo",
			expectedName:   "app_mongo_stats",
			expectedDesc:   "Response time of Mongo operations in seconds.",
		},
		{
			name:           "redis datasource",
			datasourceName: "redis",
			expectedName:   "app_redis_stats",
			expectedDesc:   "Response time of Redis operations in seconds.",
		},
		{
			name:           "arangodb datasource",
			datasourceName: "arangodb",
			expectedName:   "app_arangodb_stats",
			expectedDesc:   "Response time of Arangodb operations in seconds.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instrumenter := NewInstrumentation(tt.datasourceName)
			mockMet := newMockMetrics()
			instrumenter.SetMetrics(mockMet)

			buckets := []float64{0.1, 0.5, 1.0}
			instrumenter.RegisterStatsHistogram(buckets...)

			require.Len(t, mockMet.newHistogramCalls, 1)
			assert.Equal(t, tt.expectedName, mockMet.newHistogramCalls[0].name)
			assert.Equal(t, tt.expectedDesc, mockMet.newHistogramCalls[0].desc)
			assert.Equal(t, buckets, mockMet.newHistogramCalls[0].buckets)
		})
	}
}

func TestOperationStats(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	instrumenter := NewInstrumentation("mongo")
	mockLog := newMockLogger()
	mockMet := newMockMetrics()

	instrumenter.SetLogger(mockLog)
	instrumenter.SetMetrics(mockMet)
	instrumenter.SetTracer(tp.Tracer("test"))

	ctx := context.Background()
	query := newMockQuery("insert")
	query.metricLabels = []string{"host", "localhost", "database", "testdb"}

	// Create a span for testing
	tracerCtx, span := instrumenter.AddTrace(ctx, query)

	startTime := time.Now().Add(-100 * time.Millisecond) // Simulate 100ms operation

	instrumenter.OperationStats(tracerCtx, query, startTime, span)

	// Verify query was logged
	require.Len(t, mockLog.debugCalls, 1)
	assert.Equal(t, query, mockLog.debugCalls[0])

	// Verify duration was set on query
	assert.Positive(t, query.duration)

	// Verify histogram was recorded
	require.Len(t, mockMet.recordHistogramCalls, 1)
	assert.Equal(t, "app_mongo_stats", mockMet.recordHistogramCalls[0].name)
	assert.Equal(t, query.metricLabels, mockMet.recordHistogramCalls[0].labels)

	// Verify span was ended (check exporter has the span)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
}

func TestOperationStats_WithNilSpan(t *testing.T) {
	instrumenter := NewInstrumentation("mongo")
	mockLog := newMockLogger()
	mockMet := newMockMetrics()

	instrumenter.SetLogger(mockLog)
	instrumenter.SetMetrics(mockMet)

	ctx := context.Background()
	query := newMockQuery("query")
	startTime := time.Now().Add(-50 * time.Millisecond)

	// Should not panic with nil span
	assert.NotPanics(t, func() {
		instrumenter.OperationStats(ctx, query, startTime, nil)
	})

	// Verify logging and metrics still work
	assert.Len(t, mockLog.debugCalls, 1)
	assert.Len(t, mockMet.recordHistogramCalls, 1)
}

func TestOperationStats_DurationCalculation(t *testing.T) {
	instrumenter := NewInstrumentation("test")
	mockLog := newMockLogger()
	mockMet := newMockMetrics()

	instrumenter.SetLogger(mockLog)
	instrumenter.SetMetrics(mockMet)

	ctx := context.Background()
	query := newMockQuery("operation")
	startTime := time.Now().Add(-1 * time.Second) // 1 second ago

	instrumenter.OperationStats(ctx, query, startTime, nil)

	// Duration should be approximately 1000000 microseconds (1 second)
	assert.Greater(t, query.duration, int64(900000)) // At least 900ms
	assert.Less(t, query.duration, int64(2000000))   // Less than 2 seconds

	// Histogram value should be in seconds (approximately 1.0)
	require.Len(t, mockMet.recordHistogramCalls, 1)
	histogramValue := mockMet.recordHistogramCalls[0].value
	assert.InDelta(t, 1.0, histogramValue, 0.5) // Approximately 1 second with 0.5s tolerance
}
