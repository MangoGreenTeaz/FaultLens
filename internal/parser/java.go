package parser

import (
	"strings"

	"github.com/faultlens/faultlens/internal/model"
)

// JavaParser parses Java / Spring Boot logs and aggregates multi-line stack
// traces into a single LogEvent.
//
// A Java event starts with a "timestamp LEVEL message" line. Subsequent lines
// that are stack trace content (indented "at ..." frames, exception headers,
// "Caused by:", "Suppressed:", "... N more") are appended to the event's
// StackTrace field. A new timestamped line closes the previous event.
type JavaParser struct {
	buf *model.LogEvent
}

// NewJavaParser returns a JavaParser.
func NewJavaParser() *JavaParser { return &JavaParser{} }

// Name implements Parser.
func (*JavaParser) Name() string { return "java" }

// CanParse implements Parser: a timestamped line with a severity level, or an
// exception header line.
func (*JavaParser) CanParse(line string) bool {
	return isJavaEventStart(line) || isExceptionLine(line)
}

// Parse implements Parser.
func (p *JavaParser) Parse(line string) []*model.LogEvent {
	var out []*model.LogEvent

	switch {
	case isStackContinuation(line) || isExceptionLine(line):
		// Continuation content belongs to the buffered event, if any.
		if p.buf != nil {
			p.appendStack(line)
		}

	case isEventStart(line):
		// A new timestamped line closes the previous event.
		if p.buf != nil {
			out = append(out, p.buf)
		}
		p.buf = parseStructuredLine(line)

	default:
		// Unrecognized line: flush the buffered event. Blank lines are
		// skipped; anything else becomes a UNKNOWN-level event so nothing
		// is lost.
		if p.buf != nil {
			out = append(out, p.buf)
			p.buf = nil
		}
		if strings.TrimSpace(line) == "" {
			return out
		}
		out = append(out, &model.LogEvent{
			Raw:     line,
			Level:   model.LevelUnknown,
			Message: line,
		})
	}

	return out
}

// Flush implements Parser: returns any buffered event at end of input.
func (p *JavaParser) Flush() []*model.LogEvent {
	if p.buf == nil {
		return nil
	}
	ev := p.buf
	p.buf = nil
	return []*model.LogEvent{ev}
}

// Reset implements Parser.
func (p *JavaParser) Reset() { p.buf = nil }

// appendStack appends a stack trace line to the buffered event.
func (p *JavaParser) appendStack(line string) {
	if p.buf.StackTrace == "" {
		p.buf.StackTrace = line
		return
	}
	var b strings.Builder
	b.WriteString(p.buf.StackTrace)
	b.WriteByte('\n')
	b.WriteString(line)
	p.buf.StackTrace = b.String()
}
