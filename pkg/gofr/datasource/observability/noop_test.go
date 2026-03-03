package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNoopLogger(t *testing.T) {
	logger := NewNoopLogger()

	assert.NotNil(t, logger)
}

func TestNoopLogger_Methods(t *testing.T) {
	logger := NewNoopLogger()

	// All methods should execute without panic
	assert.NotPanics(t, func() {
		logger.Debug("test message")
	})
	assert.NotPanics(t, func() {
		logger.Debug("test", "multiple", "args")
	})
	assert.NotPanics(t, func() {
		logger.Debugf("format %s", "value")
	})
	assert.NotPanics(t, func() {
		logger.Logf("format %s %d", "value", 123)
	})
	assert.NotPanics(t, func() {
		logger.Errorf("error: %v", "something went wrong")
	})
}

func TestNewNoopMetrics(t *testing.T) {
	metrics := NewNoopMetrics()

	assert.NotNil(t, metrics)
}

func TestNoopMetrics_Methods(t *testing.T) {
	metrics := NewNoopMetrics()
	ctx := context.Background()

	// All methods should execute without panic
	assert.NotPanics(t, func() {
		metrics.NewHistogram("test_histogram", "A test histogram", 0.1, 0.5, 1.0)
	})
	assert.NotPanics(t, func() {
		metrics.NewHistogram("no_buckets", "Without buckets")
	})
	assert.NotPanics(t, func() {
		metrics.RecordHistogram(ctx, "test_histogram", 0.5, "label1", "value1")
	})
	assert.NotPanics(t, func() {
		metrics.RecordHistogram(ctx, "test_histogram", 1.0)
	})
}
