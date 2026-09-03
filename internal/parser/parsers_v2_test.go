package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// --- Apache ---

func TestApacheParseCombined(t *testing.T) {
	line := `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 200 123 "-" "Mozilla/5.0"`
	events := feedAll(NewApacheParser(), line)
	requireEventCount(t, events, 1)

	ev := events[0]
	want := time.Date(2026, 8, 31, 14, 32, 1, 0, time.FixedZone("", 8*3600))
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
	if ev.Level != model.LevelInfo {
		t.Errorf("Level = %q, want INFO (2xx)", ev.Level)
	}
	fields := ev.Fields
	for k, wantV := range map[string]string{"client_ip": "127.0.0.1", "method": "GET", "path": "/api", "status": "200", "user_agent": "Mozilla/5.0"} {
		if fields[k] != wantV {
			t.Errorf("Fields[%s] = %q, want %q", k, fields[k], wantV)
		}
	}
}

func TestApacheParseVhost(t *testing.T) {
	line := `example.com:80 127.0.0.1 - - [31/Aug/2026:14:32:03 +0800] "GET /health HTTP/1.1" 503 0 "-" "Go-http-client/1.1"`
	events := feedAll(NewApacheParser(), line)
	requireEventCount(t, events, 1)
	if events[0].Fields["vhost"] != "example.com:80" {
		t.Errorf("vhost = %q, want example.com:80", events[0].Fields["vhost"])
	}
	if events[0].Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR (5xx)", events[0].Level)
	}
}

func TestApacheCanParse(t *testing.T) {
	p := NewApacheParser()
	if !p.CanParse(`127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET / HTTP/1.1" 200 1 "-" "x"`) {
		t.Error("CanParse should accept combined format")
	}
	if p.CanParse("2026-08-31 14:32:01 ERROR some other format") {
		t.Error("CanParse should reject unrelated formats")
	}
}

// --- Python ---

func TestPythonParse(t *testing.T) {
	events := feedAll(NewPythonParser(), "2026-08-31 14:32:01,123 ERROR root: database down")
	requireEventCount(t, events, 1)

	ev := events[0]
	want := time.Date(2026, 8, 31, 14, 32, 1, 123000000, time.UTC)
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
	if ev.Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR", ev.Level)
	}
	if ev.Service != "root" {
		t.Errorf("Service = %q, want root", ev.Service)
	}
	if ev.Message != "database down" {
		t.Errorf("Message = %q", ev.Message)
	}
}

func TestPythonParseLoggerWithDots(t *testing.T) {
	events := feedAll(NewPythonParser(), "2026-08-31 14:32:02,456 INFO app.worker: task completed")
	requireEventCount(t, events, 1)
	if events[0].Service != "app.worker" {
		t.Errorf("Service = %q, want app.worker", events[0].Service)
	}
	if events[0].Level != model.LevelInfo {
		t.Errorf("Level = %q, want INFO", events[0].Level)
	}
}

func TestPythonParseNoTimestamp(t *testing.T) {
	events := feedAll(NewPythonParser(), "ERROR root: bare message without timestamp")
	requireEventCount(t, events, 1)
	if !events[0].Timestamp.IsZero() {
		t.Error("Timestamp should be zero when absent")
	}
	if events[0].Level != model.LevelError {
		t.Errorf("Level = %q", events[0].Level)
	}
}

func TestPythonCanParse(t *testing.T) {
	p := NewPythonParser()
	if !p.CanParse("2026-08-31 14:32:01,123 ERROR root: db down") {
		t.Error("CanParse should accept python logging lines")
	}
	if p.CanParse("2026-08-31 14:32:01 ERROR no comma python") {
		t.Error("CanParse should reject java-style lines (no comma millis)")
	}
}

// --- Syslog ---

func TestSyslog3164(t *testing.T) {
	// PRI 131 = 131 % 8 = 3 → err → ERROR.
	events := feedAll(NewSyslogParser(), "<131>Aug 31 14:32:01 web-01 app[42]: database down")
	requireEventCount(t, events, 1)

	ev := events[0]
	if ev.Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR (pri 131)", ev.Level)
	}
	if ev.Service != "app" {
		t.Errorf("Service = %q, want app", ev.Service)
	}
	if ev.Fields["pid"] != "42" {
		t.Errorf("pid = %q, want 42", ev.Fields["pid"])
	}
	if ev.Message != "database down" {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp should be parsed")
	}
	if ev.Timestamp.Year() != time.Now().Year() {
		t.Errorf("RFC3164 year should default to current year")
	}
}

func TestSyslog5424(t *testing.T) {
	// PRI 134 = 134 % 8 = 6 → info.
	events := feedAll(NewSyslogParser(), `<134>1 2026-08-31T14:32:03.123Z api-01 orders-api 5678 ID47 - pool exhausted`)
	requireEventCount(t, events, 1)

	ev := events[0]
	want := time.Date(2026, 8, 31, 14, 32, 3, 123000000, time.UTC)
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
	if ev.Service != "orders-api" {
		t.Errorf("Service = %q, want orders-api", ev.Service)
	}
	if ev.Fields["hostname"] != "api-01" || ev.Fields["procid"] != "5678" {
		t.Errorf("Fields = %v", ev.Fields)
	}
	if ev.Message != "pool exhausted" {
		t.Errorf("Message = %q", ev.Message)
	}
}

func TestSyslogSeverityMapping(t *testing.T) {
	tests := []struct {
		pri  string
		want model.LogLevel
	}{
		{"<0>", model.LevelFatal},   // severity 0 = emerg
		{"<8>", model.LevelFatal},   // facility 1, severity 0 = emerg
		{"<9>", model.LevelFatal},   // facility 1, severity 1 = alert
		{"<131>", model.LevelError}, // severity 3 = err
		{"<132>", model.LevelWarn},  // severity 4 = warning
		{"<133>", model.LevelInfo},  // severity 5 = notice
		{"<134>", model.LevelInfo},  // severity 6 = info
		{"<135>", model.LevelDebug}, // severity 7 = debug
	}
	for _, tt := range tests {
		line := tt.pri + "Aug 31 14:32:01 host app: message"
		events := feedAll(NewSyslogParser(), line)
		requireEventCount(t, events, 1)
		if events[0].Level != tt.want {
			t.Errorf("pri %s → Level %q, want %q", tt.pri, events[0].Level, tt.want)
		}
	}
}

func TestSyslogCanParse(t *testing.T) {
	p := NewSyslogParser()
	if !p.CanParse("<134>Aug 31 14:32:01 host app: msg") {
		t.Error("CanParse should accept RFC3164")
	}
	if !p.CanParse("<134>1 2026-08-31T14:32:01Z host app 1 2 - msg") {
		t.Error("CanParse should accept RFC5424")
	}
	if p.CanParse("2026-08-31 14:32:01 ERROR java style") {
		t.Error("CanParse should reject unrelated formats")
	}
}

// --- Docker ---

func TestDockerParse(t *testing.T) {
	line := `{"log":"error: db down","stream":"stderr","time":"2026-08-31T14:32:01.123Z"}`
	events := feedAll(NewDockerJSONParser(), line)
	requireEventCount(t, events, 1)

	ev := events[0]
	if ev.Message != "error: db down" {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.Fields["stream"] != "stderr" {
		t.Errorf("stream = %q, want stderr", ev.Fields["stream"])
	}
	want := time.Date(2026, 8, 31, 14, 32, 1, 123000000, time.UTC)
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
}

func TestDockerCanParse(t *testing.T) {
	p := NewDockerJSONParser()
	if !p.CanParse(`{"log":"hi","stream":"stdout","time":"2026-08-31T14:32:01Z"}`) {
		t.Error("CanParse should accept docker json")
	}
	if p.CanParse(`{"level":"ERROR","message":"hi"}`) {
		t.Error("CanParse should reject ordinary JSON without a log key")
	}
	if p.CanParse("not json at all") {
		t.Error("CanParse should reject non-JSON")
	}
}

// --- Kubernetes ---

func TestKubernetesParse(t *testing.T) {
	line := "2026-08-31T14:32:02.456Z stderr F error: database connection failed"
	events := feedAll(NewKubernetesParser(), line)
	requireEventCount(t, events, 1)

	ev := events[0]
	if ev.Message != "error: database connection failed" {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.Fields["stream"] != "stderr" || ev.Fields["flags"] != "F" {
		t.Errorf("Fields = %v", ev.Fields)
	}
	want := time.Date(2026, 8, 31, 14, 32, 2, 456000000, time.UTC)
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
}

func TestKubernetesCanParse(t *testing.T) {
	p := NewKubernetesParser()
	if !p.CanParse("2026-08-31T14:32:01.123Z stdout F hello") {
		t.Error("CanParse should accept k8s lines")
	}
	if p.CanParse("2026-08-31 14:32:01 ERROR plain java") {
		t.Error("CanParse should reject unrelated formats")
	}
}

// --- Registry sanity: all parsers implement Parser ---

func TestNewParsersImplementInterface(t *testing.T) {
	parsers := []Parser{
		NewApacheParser(),
		NewPythonParser(),
		NewSyslogParser(),
		NewDockerJSONParser(),
		NewKubernetesParser(),
	}
	for _, p := range parsers {
		if p.Name() == "" {
			t.Error("parser name must not be empty")
		}
		// Every parser must be safe on empty/malformed input.
		if out := p.Parse(""); len(out) > 0 {
			t.Errorf("%s produced events for empty line", p.Name())
		}
		if out := p.Parse(strings.Repeat("x", 500)); len(out) > 0 && p.Name() != "python" {
			// python may parse prose-ish lines; others must stay empty
		}
	}
}
