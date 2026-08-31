// Package parser converts raw log lines into unified model.LogEvent values.
//
// Parsers are stateful streaming consumers: lines are fed one at a time and a
// parser may buffer continuation lines (e.g. Java stack traces) until a
// complete event can be emitted. AutoParser detects the dominant format from
// content, not file extension.
package parser

import (
	"strings"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// Parser turns a stream of log lines into model.LogEvent values.
//
// Implementations are stateful: Parse feeds a single line and returns any
// events that became complete as a result. An empty result means the line did
// not produce an event (e.g. a malformed JSON line or a continuation line that
// is still buffered). Flush returns any buffered incomplete events and must be
// called at end of input.
type Parser interface {
	// Name returns the format identifier, e.g. "plain", "json", "java",
	// "nginx" or "auto".
	Name() string
	// CanParse reports whether line looks like the start of an event in
	// this format. It is used for format detection.
	CanParse(line string) bool
	// Parse feeds one raw log line and returns any completed events.
	Parse(line string) []*model.LogEvent
	// Flush returns any buffered incomplete events (call at EOF).
	Flush() []*model.LogEvent
	// Reset clears buffered state so the parser can be reused.
	Reset()
}

// Timestamp layouts tried when parsing structured log lines.
var (
	spaceTimeLayouts = []string{"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"}
	tTimeLayouts     = []string{time.RFC3339Nano, time.RFC3339}
)

// parseTimestamp extracts a leading timestamp from a log line.
//
// Supported shapes:
//
//	2026-08-31 14:32:01               (space separated)
//	2026-08-31 14:32:01.123           (space separated, fractional)
//	2026-08-31T14:32:01Z              (RFC 3339)
//	2026-08-31T14:32:01.123+08:00     (RFC 3339 with fraction and zone)
//
// It returns the parsed time, the remainder of the line after the timestamp,
// and whether a timestamp was recognized.
func parseTimestamp(line string) (time.Time, string, bool) {
	if len(line) < 19 {
		return time.Time{}, "", false
	}
	// Require a YYYY-MM-DD prefix so ordinary prose is never treated as a
	// timestamp.
	if line[4] != '-' || line[7] != '-' {
		return time.Time{}, "", false
	}

	if line[10] == 'T' {
		end := strings.IndexByte(line, ' ')
		if end < 0 {
			end = len(line)
		}
		ts := line[:end]
		for _, layout := range tTimeLayouts {
			if t, err := time.Parse(layout, ts); err == nil {
				return t, line[end:], true
			}
		}
		return time.Time{}, "", false
	}

	if line[10] != ' ' {
		return time.Time{}, "", false
	}
	rest := line[11:]
	end := strings.IndexByte(rest, ' ')
	if end < 0 {
		end = len(rest)
	}
	ts := line[:11+end]
	for _, layout := range spaceTimeLayouts {
		if t, err := time.Parse(layout, ts); err == nil {
			return t, line[11+end:], true
		}
	}
	return time.Time{}, "", false
}

// isEventStart reports whether line carries a leading timestamp and therefore
// begins a new log event.
func isEventStart(line string) bool {
	_, _, ok := parseTimestamp(line)
	return ok
}

// isJavaEventStart reports whether line begins a Java-style log event:
// a timestamp followed by a recognized severity level.
func isJavaEventStart(line string) bool {
	_, rest, ok := parseTimestamp(line)
	if !ok {
		return false
	}
	rest = strings.TrimLeft(rest, " ")
	if i := strings.IndexByte(rest, ' '); i > 0 {
		return model.ParseLevel(rest[:i]) != model.LevelUnknown
	}
	return model.ParseLevel(rest) != model.LevelUnknown
}

// parseStructuredLine parses a "timestamp LEVEL message" line into a LogEvent.
//
// A missing timestamp leaves the time zero; a missing level maps to UNKNOWN;
// the raw line is always preserved verbatim.
func parseStructuredLine(line string) *model.LogEvent {
	ev := &model.LogEvent{Raw: line, Level: model.LevelUnknown, Message: line}

	ts, rest, ok := parseTimestamp(line)
	if !ok {
		// No timestamp: keep the whole line as the message.
		return ev
	}
	ev.Timestamp = ts

	rest = strings.TrimLeft(rest, " ")
	if rest == "" {
		return ev
	}
	if i := strings.IndexByte(rest, ' '); i > 0 {
		if lvl := model.ParseLevel(rest[:i]); lvl != model.LevelUnknown {
			ev.Level = lvl
			ev.Message = strings.TrimSpace(rest[i+1:])
			return ev
		}
	}
	// Timestamp but no recognized level: the rest is the message.
	ev.Message = strings.TrimSpace(rest)
	return ev
}

// isStackContinuation reports whether line belongs to a Java stack trace:
// an indented "at ..." frame, "Caused by:", "Suppressed:", or "... N more".
func isStackContinuation(line string) bool {
	s := strings.TrimRight(line, " \t")
	if strings.HasPrefix(s, "    at ") || strings.HasPrefix(s, "\tat ") {
		return true
	}
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "... ") && strings.HasSuffix(t, " more") {
		return true
	}
	return strings.HasPrefix(t, "Caused by: ") || strings.HasPrefix(t, "Suppressed: ")
}

// isExceptionLine reports whether line is a Java exception header such as
//
//	java.sql.SQLException: Connection refused
//	com.example.UserServiceException: boom
func isExceptionLine(line string) bool {
	s := strings.TrimSpace(line)
	if i := strings.IndexByte(s, ':'); i > 0 {
		cls := s[:i]
		if strings.Contains(cls, ".") &&
			(strings.Contains(cls, "Exception") ||
				strings.Contains(cls, "Error") ||
				strings.Contains(cls, "Throwable")) {
			return true
		}
	}
	return false
}
