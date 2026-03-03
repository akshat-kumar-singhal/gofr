// Package observability provides shared interfaces and utilities for GoFr datasources.
// External datasources can import this lightweight module without depending on
// the main gofr.dev module.
package observability

// Logger interface is the common logger interface for all datasources.
// It provides a consistent logging API across all datasource implementations.
type Logger interface {
	Debug(args ...any)
	Debugf(format string, args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Logf(format string, args ...any) // alias for Infof, used by many datasources
	Error(args ...any)
	Errorf(format string, args ...any)
	Warn(args ...any)
	Warnf(format string, args ...any)
}
