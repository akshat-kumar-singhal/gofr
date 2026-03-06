package arangodb

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gofr.dev/pkg/gofr/datasource/observability"
)

func Test_PrettyPrint(t *testing.T) {
	queryLog := QueryLog{
		Query:      "",
		Duration:   12345,
		Database:   "test",
		Collection: "test",
		Filter:     true,
		ID:         "12345",
		Operation:  "getDocument",
	}
	expected := "getDocument"

	var buf bytes.Buffer

	queryLog.PrettyPrint(&buf)

	assert.Contains(t, buf.String(), expected)
}

func TestQueryLog_SetDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration int64
	}{
		{
			name:     "positive duration",
			duration: 12345,
		},
		{
			name:     "zero duration",
			duration: 0,
		},
		{
			name:     "large duration",
			duration: 9999999999,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ql := &QueryLog{}

			ql.SetDuration(tc.duration)

			assert.Equal(t, tc.duration, ql.Duration)
		})
	}
}

func TestQueryLog_GetOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
	}{
		{
			name:      "get document operation",
			operation: "getDocument",
		},
		{
			name:      "insert operation",
			operation: "insert",
		},
		{
			name:      "empty operation",
			operation: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ql := QueryLog{Operation: tc.operation}

			result := ql.GetOperation()

			assert.Equal(t, tc.operation, result)
		})
	}
}

func TestQueryLog_GetCollection(t *testing.T) {
	tests := []struct {
		name       string
		collection string
	}{
		{
			name:       "users collection",
			collection: "users",
		},
		{
			name:       "empty collection",
			collection: "",
		},
		{
			name:       "collection with special chars",
			collection: "test_collection_123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ql := QueryLog{Collection: tc.collection}

			result := ql.GetCollection()

			assert.Equal(t, tc.collection, result)
		})
	}
}

func TestQueryLog_GetTraceLabels(t *testing.T) {
	tests := []struct {
		name     string
		queryLog QueryLog
		expected map[string]string
	}{
		{
			name: "all fields populated",
			queryLog: QueryLog{
				Operation:  "getDocument",
				Database:   "testdb",
				Collection: "users",
				Graph:      "social_graph",
			},
			expected: map[string]string{
				observability.LabelOperation: "getDocument",
				observability.LabelDatabase:  "testdb",
				observability.LabelTable:     "users",
				"graph":                      "social_graph",
			},
		},
		{
			name:     "empty fields",
			queryLog: QueryLog{},
			expected: map[string]string{
				observability.LabelOperation: "",
				observability.LabelDatabase:  "",
				observability.LabelTable:     "",
				"graph":                      "",
			},
		},
		{
			name: "partial fields",
			queryLog: QueryLog{
				Operation: "insert",
				Database:  "mydb",
			},
			expected: map[string]string{
				observability.LabelOperation: "insert",
				observability.LabelDatabase:  "mydb",
				observability.LabelTable:     "",
				"graph":                      "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.queryLog.GetTraceLabels()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestQueryLog_GetMetricLabels(t *testing.T) {
	tests := []struct {
		name     string
		queryLog QueryLog
		expected []string
	}{
		{
			name: "all fields populated",
			queryLog: QueryLog{
				Operation:  "query",
				Host:       "localhost:8529",
				Database:   "testdb",
				Collection: "users",
			},
			expected: []string{
				observability.LabelOperation, "query",
				observability.LabelHost, "localhost:8529",
				observability.LabelDatabase, "testdb",
				observability.LabelTable, "users",
			},
		},
		{
			name:     "empty fields",
			queryLog: QueryLog{},
			expected: []string{
				observability.LabelOperation, "",
				observability.LabelHost, "",
				observability.LabelDatabase, "",
				observability.LabelTable, "",
			},
		},
		{
			name: "partial fields",
			queryLog: QueryLog{
				Operation: "INSERT",
				Host:      "arangodb:8529",
			},
			expected: []string{
				observability.LabelOperation, "INSERT",
				observability.LabelHost, "arangodb:8529",
				observability.LabelDatabase, "",
				observability.LabelTable, "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.queryLog.GetMetricLabels()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestQueryLog_Meta(t *testing.T) {
	tests := []struct {
		name     string
		queryLog QueryLog
		expected string
	}{
		{
			name: "all fields populated",
			queryLog: QueryLog{
				Database:   "testdb",
				Collection: "users",
				Filter:     map[string]string{"name": "John"},
				ID:         "12345",
			},
			expected: "testdb users map[name:John] 12345",
		},
		{
			name: "nil filter",
			queryLog: QueryLog{
				Database:   "testdb",
				Collection: "users",
				Filter:     nil,
				ID:         "12345",
			},
			expected: "testdb users 12345",
		},
		{
			name: "nil ID",
			queryLog: QueryLog{
				Database:   "testdb",
				Collection: "users",
				Filter:     "active=true",
				ID:         nil,
			},
			expected: "testdb users active=true",
		},
		{
			name: "nil filter and ID",
			queryLog: QueryLog{
				Database:   "testdb",
				Collection: "users",
				Filter:     nil,
				ID:         nil,
			},
			expected: "testdb users",
		},
		{
			name:     "empty values",
			queryLog: QueryLog{},
			expected: " ",
		},
		{
			name: "filter as struct",
			queryLog: QueryLog{
				Database:   "db",
				Collection: "col",
				Filter:     struct{ Key string }{Key: "value"},
			},
			expected: "db col {value}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.queryLog.Meta()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_clean(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multiple spaces",
			input:    "FOR  doc  IN  users  RETURN  doc",
			expected: "FOR doc IN users RETURN doc",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "   SELECT * FROM users   ",
			expected: "SELECT * FROM users",
		},
		{
			name:     "newlines and tabs",
			input:    "FOR doc\n\tIN users\n\tRETURN doc",
			expected: "FOR doc IN users RETURN doc",
		},
		{
			name:     "already clean string",
			input:    "FOR doc IN users RETURN doc",
			expected: "FOR doc IN users RETURN doc",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n   ",
			expected: "",
		},
		{
			name:     "mixed whitespace characters",
			input:    "  FOR\t\tdoc   \n\n IN  users  ",
			expected: "FOR doc IN users",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := clean(tc.input)

			assert.Equal(t, tc.expected, result)
		})
	}
}
