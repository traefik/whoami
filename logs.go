package main

import (
	"context"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
)

// logger is the OpenTelemetry logger used for every application and access-log
// record. It starts as the global delegate and is rebound by initLogger once the
// logger provider is configured. It is a no-op when logs are disabled.
var logger = logglobal.Logger(otelInstrumentationName)

// initLogger binds the package logger to the configured global logger provider.
func initLogger() {
	logger = logglobal.Logger(otelInstrumentationName)
}

// logInfo emits an informational log record through OpenTelemetry.
func logInfo(ctx context.Context, body string, attrs ...otellog.KeyValue) {
	emit(ctx, otellog.SeverityInfo, body, attrs...)
}

// logError emits an error log record through OpenTelemetry.
func logError(ctx context.Context, body string, attrs ...otellog.KeyValue) {
	emit(ctx, otellog.SeverityError, body, attrs...)
}

// emit builds a log record and emits it. The context carries the active span so
// records are correlated with the current trace.
func emit(ctx context.Context, severity otellog.Severity, body string, attrs ...otellog.KeyValue) {
	var record otellog.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetBody(otellog.StringValue(body))
	record.AddAttributes(attrs...)

	logger.Emit(ctx, record)
}
