package parser

import (
	"testing"

	"github.com/faultlens/faultlens/internal/model"
)

// feedAll feeds every line into p and returns all produced events including
// those released by Flush.
func feedAll(p Parser, lines ...string) []*model.LogEvent {
	var out []*model.LogEvent
	for _, l := range lines {
		out = append(out, p.Parse(l)...)
	}
	out = append(out, p.Flush()...)
	return out
}

// requireEventCount fails the test if the number of events is unexpected.
func requireEventCount(t *testing.T, events []*model.LogEvent, want int) {
	t.Helper()
	if len(events) != want {
		t.Fatalf("got %d events, want %d: %+v", len(events), want, events)
	}
}

func TestTimestampParsing(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // RFC3339 representation of the expected time
		ok   bool
	}{
		{"space separated", "2026-08-31 14:32:01 ERROR x", "2026-08-31T14:32:01Z", true},
		{"space fractional", "2026-08-31 14:32:01.123 ERROR x", "2026-08-31T14:32:01.123Z", true},
		{"rfc3339", "2026-08-31T14:32:01Z ERROR x", "2026-08-31T14:32:01Z", true},
		{"rfc3339 nano", "2026-08-31T14:32:01.123+08:00 ERROR x", "2026-08-31T06:32:01.123Z", true},
		{"no timestamp", "just prose", "", false},
		{"not a date", "foo-bar-baz ERROR x", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, ok := parseTimestamp(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got.UTC().Format("2006-01-02T15:04:05.999999999Z07:00") != tt.want {
				t.Errorf("parsed %v, want %s", got, tt.want)
			}
		})
	}
}

func TestIsStackContinuation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"indented at", "    at com.example.Foo.bar(Foo.java:1)", true},
		{"tab at", "\tat com.example.Foo.bar(Foo.java:1)", true},
		{"caused by", "Caused by: java.lang.RuntimeException: boom", true},
		{"suppressed", "Suppressed: java.io.IOException: closed", true},
		{"n more", "    ... 3 more", true},
		{"ordinary line", "plain message without indent", false},
		{"unindented at", "at com.example.Foo.bar(Foo.java:1)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStackContinuation(tt.in); got != tt.want {
				t.Errorf("isStackContinuation(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsExceptionLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"jdbc exception", "java.sql.SQLException: Connection refused", true},
		{"custom exception", "com.example.UserServiceException: boom", true},
		{"out of memory", "java.lang.OutOfMemoryError: Java heap space", true},
		{"caused by header", "Caused by: java.net.ConnectException: refused", false},
		{"plain message with colon", "ERROR connection refused: yes", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExceptionLine(tt.in); got != tt.want {
				t.Errorf("isExceptionLine(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
