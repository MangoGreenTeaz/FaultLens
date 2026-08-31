// Package model defines the shared data types used across FaultLens.
//
// Every parser converts its native log format into a LogEvent; all downstream
// packages (grouping, timeline, anomaly, diagnosis, output) only ever see this
// unified shape.
package model

import (
	"strings"
	"time"
)

// LogLevel represents the severity of a log event.
type LogLevel string

// Supported log levels. Parser output is normalized to one of these values.
const (
	LevelTrace   LogLevel = "TRACE"
	LevelDebug   LogLevel = "DEBUG"
	LevelInfo    LogLevel = "INFO"
	LevelWarn    LogLevel = "WARN"
	LevelError   LogLevel = "ERROR"
	LevelFatal   LogLevel = "FATAL"
	LevelUnknown LogLevel = "UNKNOWN"
)

// ParseLevel normalizes a level string into a LogLevel.
//
// Matching is case-insensitive and tolerant of surrounding whitespace.
// Common aliases (WARNING, ERR, CRITICAL, PANIC) are folded into the nearest
// canonical level. Unrecognized values map to LevelUnknown instead of failing.
func ParseLevel(s string) LogLevel {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "TRACE":
		return LevelTrace
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR", "ERR":
		return LevelError
	case "FATAL", "CRITICAL", "CRIT", "PANIC":
		return LevelFatal
	default:
		return LevelUnknown
	}
}

// LogEvent is the unified representation of a single parsed log entry.
type LogEvent struct {
	// Timestamp is the event time. The zero value means the log line did not
	// carry a parseable timestamp.
	Timestamp time.Time
	// Level is the severity of the event, defaulting to LevelUnknown.
	Level LogLevel
	// Service is the emitting service name when known.
	Service string
	// Message is the human-readable message.
	Message string
	// Source identifies where the event came from (e.g. file name or stdin).
	Source string
	// Fields holds extra key/value pairs extracted by the parser
	// (e.g. Nginx status/method/path, JSON extra fields).
	Fields map[string]string
	// Raw is the original log line exactly as read from the input.
	Raw string
	// StackTrace holds the full multi-line stack trace for Java-like
	// exceptions; empty otherwise.
	StackTrace string
}
