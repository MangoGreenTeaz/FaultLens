package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// Python standard logging format: "2026-08-31 14:32:01,123 ERROR root: message".
// The fractional seconds use a comma, unlike Go's native dot notation.
var pythonRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[,.]\d+)?\s*(\w+)\s+([\w.]+):\s*(.*)$`)

// PythonParser parses Python logging output.
type PythonParser struct{}

// NewPythonParser returns a PythonParser.
func NewPythonParser() *PythonParser { return &PythonParser{} }

// Name implements Parser.
func (*PythonParser) Name() string { return "python" }

// CanParse implements Parser.
func (*PythonParser) CanParse(line string) bool {
	return pythonRe.MatchString(line)
}

// Parse implements Parser.
func (*PythonParser) Parse(line string) []*model.LogEvent {
	m := pythonRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}

	ev := &model.LogEvent{Raw: line, Level: model.LevelUnknown, Message: line}

	if m[1] != "" {
		// Convert the comma fractional seconds to a dot for time.Parse.
		ts := strings.Replace(m[1], ",", ".", 1)
		if t, err := time.Parse("2006-01-02 15:04:05.999999999", ts); err == nil {
			ev.Timestamp = t
		}
	}
	ev.Level = model.ParseLevel(m[2])
	ev.Service = m[3]
	ev.Message = m[4]
	return []*model.LogEvent{ev}
}

// Flush implements Parser. Python parsing is stateless.
func (*PythonParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (*PythonParser) Issues() int { return 0 }

// Reset implements Parser.
func (*PythonParser) Reset() {}
