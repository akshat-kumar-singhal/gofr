package mongo

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// QueryLog represents a MongoDB query log entry for debugging and monitoring.
type QueryLog struct {
	Query      string `json:"query"`
	Duration   int64  `json:"duration"`
	Collection string `json:"collection,omitempty"`
	Filter     any    `json:"filter,omitempty"`
	ID         any    `json:"id,omitempty"`
	Update     any    `json:"update,omitempty"`
}

// SetDuration sets the duration field (implements observability.QueryLogger).
func (ql *QueryLog) SetDuration(d int64) {
	ql.Duration = d
}

func (ql *QueryLog) GetOperation() string {
	return ql.Query
}

func (ql *QueryLog) PrettyPrint(writer io.Writer) {
	if ql.Filter == nil {
		ql.Filter = ""
	}

	if ql.ID == nil {
		ql.ID = ""
	}

	if ql.Update == nil {
		ql.Update = ""
	}

	fmt.Fprintf(writer, "\u001B[38;5;8m%-32s \u001B[38;5;206m%-6s\u001B[0m %8d\u001B[38;5;8mµs\u001B[0m %s\n",
		clean(ql.Query), "MONGO", ql.Duration, ql.Meta())
}

// clean takes a string query as input and performs two operations to clean it up:
// 1. It replaces multiple consecutive whitespace characters with a single space.
// 2. It trims leading and trailing whitespace from the string.
// The cleaned-up query string is then returned.
func clean(query string) string {
	// Replace multiple consecutive whitespace characters with a single space
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")

	// Trim leading and trailing whitespace from the string
	query = strings.TrimSpace(query)

	return query
}

func (ql *QueryLog) Meta() string {
	list := []string{ql.Collection}

	if ql.Filter != nil {
		list = append(list, fmt.Sprintf("%v", ql.Filter))
	}

	if ql.ID != nil {
		list = append(list, fmt.Sprintf("%v", ql.ID))
	}

	if ql.Update != nil {
		list = append(list, fmt.Sprintf("%v", ql.Update))
	}

	return strings.Join(list, " ")
}
