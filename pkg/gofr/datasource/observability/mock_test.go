package observability

import "context"

// mockQuery implements ObservableQuery for testing.
type mockQuery struct {
	operation    string
	duration     int64
	metricLabels []string
	traceLabels  map[string]string
}

func newMockQuery(operation string) *mockQuery {
	return &mockQuery{
		operation:    operation,
		metricLabels: []string{},
		traceLabels:  make(map[string]string),
	}
}

func (m *mockQuery) SetDuration(d int64) {
	m.duration = d
}

func (m *mockQuery) GetOperation() string {
	return m.operation
}

func (m *mockQuery) GetMetricLabels() []string {
	return m.metricLabels
}

func (m *mockQuery) GetTraceLabels() map[string]string {
	return m.traceLabels
}

// mockLogger implements Logger for testing and records calls.
type mockLogger struct {
	debugCalls  []any
	debugfCalls []mockLogCall
	logfCalls   []mockLogCall
	errorfCalls []mockLogCall
}

type mockLogCall struct {
	format string
	args   []any
}

func newMockLogger() *mockLogger {
	return &mockLogger{}
}

func (m *mockLogger) Debug(args ...any) {
	m.debugCalls = append(m.debugCalls, args...)
}

func (m *mockLogger) Debugf(format string, args ...any) {
	m.debugfCalls = append(m.debugfCalls, mockLogCall{format: format, args: args})
}

func (m *mockLogger) Logf(format string, args ...any) {
	m.logfCalls = append(m.logfCalls, mockLogCall{format: format, args: args})
}

func (m *mockLogger) Errorf(format string, args ...any) {
	m.errorfCalls = append(m.errorfCalls, mockLogCall{format: format, args: args})
}

// mockMetrics implements Metrics for testing and records calls.
type mockMetrics struct {
	newHistogramCalls    []mockHistogramCall
	recordHistogramCalls []mockRecordCall
}

type mockHistogramCall struct {
	name    string
	desc    string
	buckets []float64
}

type mockRecordCall struct {
	ctx    context.Context
	name   string
	value  float64
	labels []string
}

func newMockMetrics() *mockMetrics {
	return &mockMetrics{}
}

func (m *mockMetrics) NewHistogram(name, desc string, buckets ...float64) {
	m.newHistogramCalls = append(m.newHistogramCalls, mockHistogramCall{
		name:    name,
		desc:    desc,
		buckets: buckets,
	})
}

func (m *mockMetrics) RecordHistogram(ctx context.Context, name string, value float64, labels ...string) {
	m.recordHistogramCalls = append(m.recordHistogramCalls, mockRecordCall{
		ctx:    ctx,
		name:   name,
		value:  value,
		labels: labels,
	})
}
