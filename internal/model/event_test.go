package model

import "testing"

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want LogLevel
	}{
		{"trace", "TRACE", LevelTrace},
		{"trace lowercase", "trace", LevelTrace},
		{"debug", "DEBUG", LevelDebug},
		{"info", "INFO", LevelInfo},
		{"info mixed case", "Info", LevelInfo},
		{"warn", "WARN", LevelWarn},
		{"warning alias", "WARNING", LevelWarn},
		{"error", "ERROR", LevelError},
		{"err alias", "ERR", LevelError},
		{"fatal", "FATAL", LevelFatal},
		{"critical alias", "CRITICAL", LevelFatal},
		{"panic alias", "PANIC", LevelFatal},
		{"surrounding whitespace", "  error  ", LevelError},
		{"empty string", "", LevelUnknown},
		{"unknown value", "not-a-level", LevelUnknown},
		{"numeric string", "500", LevelUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseLevel(tt.in); got != tt.want {
				t.Errorf("ParseLevel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLogEventZeroValues(t *testing.T) {
	// A zero-value LogEvent must be usable without panicking. Its Level is
	// the empty string (Go zero value); parsers are responsible for mapping
	// missing levels to LevelUnknown via ParseLevel.
	e := LogEvent{}
	if e.Level != "" {
		t.Errorf("zero LogEvent level = %q, want empty string (Go zero value)", e.Level)
	}
	if e.Message != "" {
		t.Errorf("zero LogEvent message = %q, want empty", e.Message)
	}
	if len(e.Fields) != 0 {
		t.Errorf("zero LogEvent fields = %v, want nil", e.Fields)
	}
}

func TestParseLevelMapsEmptyToUnknown(t *testing.T) {
	// The parser-facing normalization path: an empty level string must
	// become UNKNOWN, never an empty LogLevel.
	if got := ParseLevel(""); got != LevelUnknown {
		t.Errorf("ParseLevel(\"\") = %q, want UNKNOWN", got)
	}
}
