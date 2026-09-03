package parser

import (
	"strings"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// PlainTextParser parses simple "timestamp LEVEL message" lines.
//
// It is the fallback format: any line that cannot be recognized as JSON,
// Java or Nginx is handled here without failing. Unrecognized structure
// produces a UNKNOWN-level event carrying the raw line.
type PlainTextParser struct{}

// NewPlainTextParser returns a PlainTextParser.
func NewPlainTextParser() *PlainTextParser { return &PlainTextParser{} }

// Name implements Parser.
func (*PlainTextParser) Name() string { return "plain" }

// CanParse implements Parser. Plain text accepts any non-empty line.
func (*PlainTextParser) CanParse(line string) bool {
	return len(line) > 0
}

// Parse implements Parser. Blank lines are skipped so they do not pollute
// event statistics.
func (*PlainTextParser) Parse(line string) []*model.LogEvent {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	return []*model.LogEvent{parseStructuredLine(line)}
}

// Flush implements Parser. Plain parsing is stateless.
func (*PlainTextParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser. Plain text never fails.
func (*PlainTextParser) Issues() int { return 0 }

// Reset implements Parser.
func (*PlainTextParser) Reset() {}
