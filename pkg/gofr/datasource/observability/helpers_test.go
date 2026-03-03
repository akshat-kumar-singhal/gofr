package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase word",
			input:    "mongo",
			expected: "Mongo",
		},
		{
			name:     "already capitalized",
			input:    "Mongo",
			expected: "Mongo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character lowercase",
			input:    "a",
			expected: "A",
		},
		{
			name:     "single character uppercase",
			input:    "A",
			expected: "A",
		},
		{
			name:     "multiple words - only first capitalized",
			input:    "hello world",
			expected: "Hello world",
		},
		{
			name:     "all uppercase",
			input:    "REDIS",
			expected: "REDIS",
		},
		{
			name:     "mixed case",
			input:    "aRaNgOdB",
			expected: "ARaNgOdB",
		},
		{
			name:     "with numbers",
			input:    "redis6",
			expected: "Redis6",
		},
		{
			name:     "starts with number",
			input:    "6redis",
			expected: "6redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := titleCase(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatsHistogramName(t *testing.T) {
	tests := []struct {
		name           string
		datasourceName string
		expected       string
	}{
		{
			name:           "mongo",
			datasourceName: "mongo",
			expected:       "app_mongo_stats",
		},
		{
			name:           "redis",
			datasourceName: "redis",
			expected:       "app_redis_stats",
		},
		{
			name:           "arangodb",
			datasourceName: "arangodb",
			expected:       "app_arangodb_stats",
		},
		{
			name:           "empty name",
			datasourceName: "",
			expected:       "app__stats",
		},
		{
			name:           "with underscore",
			datasourceName: "my_custom",
			expected:       "app_my_custom_stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &instrumentation{datasourceName: tt.datasourceName}
			result := inst.statsHistogramName()

			assert.Equal(t, tt.expected, result)
		})
	}
}
