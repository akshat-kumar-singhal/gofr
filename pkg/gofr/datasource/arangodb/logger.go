package arangodb

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"gofr.dev/pkg/gofr/datasource/observability"
)

// QueryLog represents an ArangoDB query log entry for debugging and monitoring.
type QueryLog struct {
	Query      string `json:"query"`
	Duration   int64  `json:"duration"`
	Host       string `json:"host,omitempty"`
	Database   string `json:"database,omitempty"`
	Graph      string `json:"graph,omitempty"`
	Collection string `json:"collection,omitempty"`
	Filter     any    `json:"filter,omitempty"`
	ID         any    `json:"id,omitempty"`
	Operation  string `json:"operation,omitempty"`
}

// SetDuration sets the duration field (implements observability.QueryLogger).
func (ql *QueryLog) SetDuration(d int64) {
	ql.Duration = d
}

func (ql *QueryLog) GetOperation() string {
	return ql.Operation
}

func (ql *QueryLog) GetCollection() string {
	return ql.Collection
}

func (ql *QueryLog) GetTraceLabels() map[string]string {
	return map[string]string{
		observability.LabelOperation: ql.Operation,
		observability.LabelDatabase:  ql.Database,
		observability.LabelTable:     ql.Collection,
		"graph":                      ql.Graph,
	}
}

func (ql *QueryLog) GetMetricLabels() []string {
	return []string{
		observability.LabelOperation, ql.Query,
		observability.LabelHost, ql.Host,
		observability.LabelDatabase, ql.Database,
		observability.LabelTable, ql.Collection,
	}
}

// PrettyPrint formats the QueryLog for output.
func (ql *QueryLog) PrettyPrint(writer io.Writer) {
	if ql.Filter == nil {
		ql.Filter = ""
	}

	if ql.ID == nil {
		ql.ID = ""
	}

	fmt.Fprintf(writer, "\u001B[38;5;8m%-32s \u001B[38;5;206m%-6s\u001B[0m %8d\u001B[38;5;8mµs\u001B[0m %s %s\n",
		clean(ql.Operation), "ARANGODB", ql.Duration, ql.Meta(), clean(ql.Query))
}

func clean(query string) string {
	// Replace multiple consecutive whitespace characters with a single space
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")
	// Trim leading and trailing whitespace from the string
	return strings.TrimSpace(query)
}

func (ql *QueryLog) Meta() string {
	list := []string{ql.Database, ql.Collection}

	if ql.Filter != nil {
		list = append(list, fmt.Sprintf("%v", ql.Filter))
	}

	if ql.ID != nil {
		list = append(list, fmt.Sprintf("%v", ql.ID))
	}

	return strings.Join(list, " ")
}
