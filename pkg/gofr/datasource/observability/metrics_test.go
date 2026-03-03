package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultHistogramBuckets(t *testing.T) {
	buckets := GetDefaultHistogramBuckets()

	// Should return 16 buckets
	require.Len(t, buckets, 16)

	// First bucket should be 0.05 (50ms)
	assert.Equal(t, 0.05, buckets[0])

	// Last bucket should be 10 (10 seconds)
	assert.Equal(t, 10.0, buckets[len(buckets)-1])
}

func TestGetDefaultHistogramBuckets_AscendingOrder(t *testing.T) {
	buckets := GetDefaultHistogramBuckets()

	for i := 1; i < len(buckets); i++ {
		assert.Greater(t, buckets[i], buckets[i-1],
			"bucket at index %d should be greater than bucket at index %d", i, i-1)
	}
}

func TestGetDefaultHistogramBuckets_ExpectedValues(t *testing.T) {
	buckets := GetDefaultHistogramBuckets()
	expected := []float64{.05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}

	assert.Equal(t, expected, buckets)
}

func TestMetricLabelConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "LabelHost",
			constant: LabelHost,
			expected: "host",
		},
		{
			name:     "LabelDatabase",
			constant: LabelDatabase,
			expected: "database",
		},
		{
			name:     "LabelOperation",
			constant: LabelOperation,
			expected: "operation",
		},
		{
			name:     "LabelTable",
			constant: LabelTable,
			expected: "table",
		},
		{
			name:     "LabelBucket",
			constant: LabelBucket,
			expected: "bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}
