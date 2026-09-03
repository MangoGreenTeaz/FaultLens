package parser

import (
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

func TestJSONParseBasic(t *testing.T) {
	p := NewJSONParser()
	events := feedAll(p, `{"timestamp":"2026-08-31T14:32:01Z","level":"ERROR","message":"database connection failed","service":"api"}`)
	requireEventCount(t, events, 1)

	ev := events[0]
	wantTS := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	if !ev.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, wantTS)
	}
	if ev.Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR", ev.Level)
	}
	if ev.Message != "database connection failed" {
		t.Errorf("Message = %q", ev.Message)
	}
	if ev.Service != "api" {
		t.Errorf("Service = %q, want api", ev.Service)
	}
	if ev.Fields != nil {
		t.Errorf("Fields = %v, want nil", ev.Fields)
	}
}

func TestJSONParseAliases(t *testing.T) {
	p := NewJSONParser()
	events := feedAll(p, `{"ts":"2026-08-31T14:32:01Z","severity":"warn","msg":"cache miss","service_name":"worker"}`)
	requireEventCount(t, events, 1)

	if events[0].Level != model.LevelWarn {
		t.Errorf("Level = %q, want WARN", events[0].Level)
	}
	if events[0].Message != "cache miss" {
		t.Errorf("Message = %q", events[0].Message)
	}
	if events[0].Service != "worker" {
		t.Errorf("Service = %q", events[0].Service)
	}
}

func TestJSONParseUnknownFields(t *testing.T) {
	p := NewJSONParser()
	events := feedAll(p, `{"level":"INFO","message":"request handled","request_id":"abc-123","user_id":42,"nested":{"a":1}}`)
	requireEventCount(t, events, 1)

	fields := events[0].Fields
	if fields["request_id"] != "abc-123" {
		t.Errorf("Fields[request_id] = %q, want abc-123", fields["request_id"])
	}
	if fields["user_id"] != "42" {
		t.Errorf("Fields[user_id] = %q, want 42", fields["user_id"])
	}
	if fields["nested"] == "" {
		t.Error("nested field should be serialized")
	}
	if _, ok := fields["message"]; ok {
		t.Error("known field message leaked into Fields")
	}
}

func TestJSONParseMalformed(t *testing.T) {
	p := NewJSONParser()
	events := feedAll(p, "this is not json at all")
	requireEventCount(t, events, 0)
}

func TestJSONParseMissingFields(t *testing.T) {
	p := NewJSONParser()
	events := feedAll(p, `{"some_field":"value"}`)
	requireEventCount(t, events, 1)
	ev := events[0]
	if !ev.Timestamp.IsZero() {
		t.Errorf("Timestamp = %v, want zero", ev.Timestamp)
	}
	if ev.Level != model.LevelUnknown {
		t.Errorf("Level = %q, want UNKNOWN", ev.Level)
	}
	if ev.Fields["some_field"] != "value" {
		t.Errorf("Fields[some_field] = %q", ev.Fields["some_field"])
	}
}

func TestJSONCanParse(t *testing.T) {
	p := NewJSONParser()
	if !p.CanParse(`{"a":1}`) {
		t.Error("CanParse should accept object")
	}
	if p.CanParse("plain text") {
		t.Error("CanParse should reject plain text")
	}
}
