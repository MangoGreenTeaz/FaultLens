package parser

import (
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

func TestPlainParseBasic(t *testing.T) {
	p := NewPlainTextParser()
	events := feedAll(p, "2026-08-31 14:32:01 ERROR database connection failed")
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
		t.Errorf("Message = %q, want %q", ev.Message, "database connection failed")
	}
	if ev.Raw != "2026-08-31 14:32:01 ERROR database connection failed" {
		t.Errorf("Raw was not preserved")
	}
}

func TestPlainParseRFC3339(t *testing.T) {
	p := NewPlainTextParser()
	events := feedAll(p, "2026-08-31T14:32:01Z ERROR database connection failed")
	requireEventCount(t, events, 1)
	if !events[0].Timestamp.Equal(time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)) {
		t.Errorf("Timestamp = %v", events[0].Timestamp)
	}
	if events[0].Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR", events[0].Level)
	}
}

func TestPlainParseNoLevel(t *testing.T) {
	p := NewPlainTextParser()
	events := feedAll(p, "2026-08-31 14:32:01 user logged in")
	requireEventCount(t, events, 1)
	if !events[0].Timestamp.Equal(time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)) {
		t.Errorf("Timestamp = %v", events[0].Timestamp)
	}
	if events[0].Level != model.LevelUnknown {
		t.Errorf("Level = %q, want UNKNOWN", events[0].Level)
	}
	if events[0].Message != "user logged in" {
		t.Errorf("Message = %q, want %q", events[0].Message, "user logged in")
	}
}

func TestPlainParseNoTimestamp(t *testing.T) {
	p := NewPlainTextParser()
	events := feedAll(p, "a line without any timestamp")
	requireEventCount(t, events, 1)
	if !events[0].Timestamp.IsZero() {
		t.Errorf("Timestamp = %v, want zero", events[0].Timestamp)
	}
	if events[0].Level != model.LevelUnknown {
		t.Errorf("Level = %q, want UNKNOWN", events[0].Level)
	}
	if events[0].Message != "a line without any timestamp" {
		t.Errorf("Message = %q, want raw line", events[0].Message)
	}
}

func TestPlainParseFractionalTimestamp(t *testing.T) {
	p := NewPlainTextParser()
	events := feedAll(p, "2026-08-31 14:32:01.123 WARN slow request")
	requireEventCount(t, events, 1)
	want := time.Date(2026, 8, 31, 14, 32, 1, 123000000, time.UTC)
	if !events[0].Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", events[0].Timestamp, want)
	}
	if events[0].Level != model.LevelWarn {
		t.Errorf("Level = %q, want WARN", events[0].Level)
	}
}

func TestPlainSkipBlankLines(t *testing.T) {
	p := NewPlainTextParser()
	events := feedAll(p, "", "   ", "2026-08-31 14:32:01 ERROR x", "")
	requireEventCount(t, events, 1)
}

func TestPlainCanParse(t *testing.T) {
	p := NewPlainTextParser()
	if !p.CanParse("anything") {
		t.Error("plain CanParse should accept any line")
	}
}
