package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

func TestJavaParseStackTraceAggregated(t *testing.T) {
	p := NewJavaParser()
	events := feedAll(p,
		"2026-08-31 14:32:01 ERROR Database query failed",
		"java.sql.SQLException: Connection refused",
		"    at com.example.UserService.query(UserService.java:182)",
		"    at com.example.UserController.get(UserController.java:71)",
		"    ... 3 more",
		"Caused by: java.net.ConnectException: Connection refused",
		"    at java.base/sun.nio.ch.Net.connect0(Native Method)",
	)
	// Everything is buffered into ONE event; nothing emitted before Flush.
	requireEventCount(t, events, 1)

	ev := events[0]
	wantTS := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	if !ev.Timestamp.Equal(wantTS) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, wantTS)
	}
	if ev.Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR", ev.Level)
	}
	if ev.Message != "Database query failed" {
		t.Errorf("Message = %q, want %q", ev.Message, "Database query failed")
	}
	stack := ev.StackTrace
	for _, want := range []string{
		"java.sql.SQLException: Connection refused",
		"at com.example.UserService.query(UserService.java:182)",
		"at com.example.UserController.get(UserController.java:71)",
		"at java.base/sun.nio.ch.Net.connect0(Native Method)",
	} {
		if !strings.Contains(stack, want) {
			t.Errorf("StackTrace missing %q; got:\n%s", want, stack)
		}
	}
}

func TestJavaParseMultipleEvents(t *testing.T) {
	p := NewJavaParser()
	events := feedAll(p,
		"2026-08-31 14:32:01 ERROR Database query failed",
		"java.sql.SQLException: Connection refused",
		"    at com.example.UserService.query(UserService.java:182)",
		"2026-08-31 14:32:02 ERROR Second failure",
	)
	requireEventCount(t, events, 2)

	// First event closes when the second timestamped line arrives.
	if events[0].Timestamp.Hour() != 14 || events[0].Timestamp.Minute() != 32 || events[0].Timestamp.Second() != 1 {
		t.Errorf("first event Timestamp = %v", events[0].Timestamp)
	}
	if !strings.Contains(events[0].StackTrace, "SQLException") {
		t.Errorf("first event should carry its stack trace; got:\n%s", events[0].StackTrace)
	}
	if events[1].Message != "Second failure" {
		t.Errorf("second event Message = %q", events[1].Message)
	}
	if events[1].StackTrace != "" {
		t.Errorf("second event should have no stack trace, got %q", events[1].StackTrace)
	}
}

func TestJavaParseFlushAtEOF(t *testing.T) {
	p := NewJavaParser()
	// The stack trace continues to the end of input; Flush must release it.
	_ = p.Parse("2026-08-31 14:32:01 ERROR boom")
	_ = p.Parse("java.lang.RuntimeException: boom")
	_ = p.Parse("    at com.example.Main.main(Main.java:10)")
	events := p.Flush()
	requireEventCount(t, events, 1)
	if !strings.Contains(events[0].StackTrace, "Main.java:10") {
		t.Errorf("Flush should release the buffered stack trace")
	}
}

func TestJavaParseSuppressed(t *testing.T) {
	p := NewJavaParser()
	events := feedAll(p,
		"2026-08-31 14:32:01 ERROR resource leak",
		"java.io.IOException: stream closed",
		"    at com.example.FileReader.read(FileReader.java:42)",
		"Suppressed: java.io.IOException: failed to delete temp file",
		"    at com.example.FileReader.cleanup(FileReader.java:88)",
	)
	requireEventCount(t, events, 1)
	stack := events[0].StackTrace
	if !strings.Contains(stack, "Suppressed: java.io.IOException") {
		t.Errorf("Suppressed block missing; got:\n%s", stack)
	}
}

func TestJavaParseSpringBootTimestamp(t *testing.T) {
	p := NewJavaParser()
	events := feedAll(p, "2026-08-31T14:32:01.123+08:00 ERROR Boot application failed to start")
	requireEventCount(t, events, 1)

	want := time.Date(2026, 8, 31, 14, 32, 1, 123000000, time.FixedZone("", 8*3600))
	if !events[0].Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", events[0].Timestamp, want)
	}
	if events[0].Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR", events[0].Level)
	}
}

func TestJavaParseUnrecognizedLine(t *testing.T) {
	p := NewJavaParser()
	events := feedAll(p, "some random non-log text")
	requireEventCount(t, events, 1)
	if events[0].Level != model.LevelUnknown {
		t.Errorf("Level = %q, want UNKNOWN", events[0].Level)
	}
	if events[0].Message != "some random non-log text" {
		t.Errorf("Message = %q, want raw line", events[0].Message)
	}
}

func TestJavaParseBlankLineBetweenEvents(t *testing.T) {
	p := NewJavaParser()
	events := feedAll(p,
		"2026-08-31 14:32:01 ERROR first",
		"java.lang.RuntimeException: one",
		"    at com.example.A.a(A.java:1)",
		"",
		"2026-08-31 14:32:02 ERROR second",
	)
	requireEventCount(t, events, 2)
	if !strings.Contains(events[0].StackTrace, "RuntimeException") {
		t.Errorf("blank line must not drop the buffered stack trace")
	}
	if events[1].Message != "second" {
		t.Errorf("second event Message = %q", events[1].Message)
	}
}

func TestJavaCanParse(t *testing.T) {
	p := NewJavaParser()
	if !p.CanParse("2026-08-31 14:32:01 ERROR database down") {
		t.Error("CanParse should accept a timestamped ERROR line")
	}
	if !p.CanParse("java.sql.SQLException: Connection refused") {
		t.Error("CanParse should accept an exception header line")
	}
	if p.CanParse("    at com.example.Foo.bar(Foo.java:1)") {
		t.Error("CanParse should reject a stack frame (continuation, not a start)")
	}
	if p.CanParse("hello world") {
		t.Error("CanParse should reject prose")
	}
}

func TestJavaParserReset(t *testing.T) {
	p := NewJavaParser()
	_ = p.Parse("2026-08-31 14:32:01 ERROR first")
	p.Reset()
	events := p.Flush()
	requireEventCount(t, events, 0)
}
