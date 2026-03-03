package mongo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"gofr.dev/pkg/gofr/datasource/observability"
)

func TestLoggingDataPresent(t *testing.T) {
	queryLog := QueryLog{
		Query:      "find",
		Duration:   12345,
		Collection: "users",
		Filter:     map[string]string{"name": "John"},
		ID:         "123",
		Update:     map[string]string{"$set": "Doe"},
	}
	expected := "name:John"

	var buf bytes.Buffer

	queryLog.PrettyPrint(&buf)

	assert.Contains(t, buf.String(), expected)
}

func TestLoggingEmptyData(t *testing.T) {
	queryLog := QueryLog{
		Query:    "insert",
		Duration: 6789,
	}
	expected := "name:John"

	var buf bytes.Buffer

	queryLog.PrettyPrint(&buf)

	assert.NotContains(t, buf.String(), expected)
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
		name  string
		query string
	}{
		{
			name:  "find operation",
			query: "find",
		},
		{
			name:  "insert operation",
			query: "insertOne",
		},
		{
			name:  "empty query",
			query: "",
		},
		{
			name:  "aggregate operation",
			query: "aggregate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ql := QueryLog{Query: tc.query}

			result := ql.GetOperation()

			assert.Equal(t, tc.query, result)
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
				Query:      "find",
				Database:   "testdb",
				Collection: "users",
			},
			expected: map[string]string{
				observability.LabelOperation: "find",
				observability.LabelDatabase:  "testdb",
				observability.LabelTable:     "users",
			},
		},
		{
			name:     "empty fields",
			queryLog: QueryLog{},
			expected: map[string]string{
				observability.LabelOperation: "",
				observability.LabelDatabase:  "",
				observability.LabelTable:     "",
			},
		},
		{
			name: "partial fields",
			queryLog: QueryLog{
				Query:    "insertOne",
				Database: "mydb",
			},
			expected: map[string]string{
				observability.LabelOperation: "insertOne",
				observability.LabelDatabase:  "mydb",
				observability.LabelTable:     "",
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
				Query:      "find",
				Host:       "localhost:27017",
				Database:   "testdb",
				Collection: "users",
			},
			expected: []string{
				observability.LabelOperation, "find",
				observability.LabelHost, "localhost:27017",
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
				Query: "insertOne",
				Host:  "mongodb:27017",
			},
			expected: []string{
				observability.LabelOperation, "insertOne",
				observability.LabelHost, "mongodb:27017",
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
				Collection: "users",
				Filter:     map[string]string{"name": "John"},
				ID:         "12345",
				Update:     map[string]string{"$set": "Doe"},
			},
			expected: "users map[name:John] 12345 map[$set:Doe]",
		},
		{
			name: "nil filter",
			queryLog: QueryLog{
				Collection: "users",
				Filter:     nil,
				ID:         "12345",
				Update:     "update_value",
			},
			expected: "users 12345 update_value",
		},
		{
			name: "nil ID",
			queryLog: QueryLog{
				Collection: "users",
				Filter:     "active=true",
				ID:         nil,
				Update:     "update",
			},
			expected: "users active=true update",
		},
		{
			name: "nil update",
			queryLog: QueryLog{
				Collection: "users",
				Filter:     "filter",
				ID:         "id",
				Update:     nil,
			},
			expected: "users filter id",
		},
		{
			name: "all nil optional fields",
			queryLog: QueryLog{
				Collection: "users",
				Filter:     nil,
				ID:         nil,
				Update:     nil,
			},
			expected: "users",
		},
		{
			name:     "empty values",
			queryLog: QueryLog{},
			expected: "",
		},
		{
			name: "filter as struct",
			queryLog: QueryLog{
				Collection: "col",
				Filter:     struct{ Key string }{Key: "value"},
			},
			expected: "col {value}",
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
			input:    "db.users.find(  {  }  )",
			expected: "db.users.find( { } )",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "   db.users.find()   ",
			expected: "db.users.find()",
		},
		{
			name:     "newlines and tabs",
			input:    "db.users\n\t.find()\n\t.limit(10)",
			expected: "db.users .find() .limit(10)",
		},
		{
			name:     "already clean string",
			input:    "db.users.find()",
			expected: "db.users.find()",
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
			input:    "  db.users\t\t.find(   \n\n {  })  ",
			expected: "db.users .find( { })",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := clean(tc.input)

			assert.Equal(t, tc.expected, result)
		})
	}
}
