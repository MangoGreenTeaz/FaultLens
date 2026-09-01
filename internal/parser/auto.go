package parser

import (
	"encoding/json"
	"strings"

	"github.com/faultlens/faultlens/internal/model"
)

// sniffLimit is how many leading lines are buffered before AutoParser commits
// to a concrete format.
const sniffLimit = 20

// AutoParser detects the dominant log format from content and delegates to
// the matching parser. Detection runs on the first sniffLimit lines; format
// detection order is JSON → Java → Nginx → Plain.
type AutoParser struct {
	active  Parser
	pending []string
	output  []*model.LogEvent
}

// NewAutoParser returns an AutoParser.
func NewAutoParser() *AutoParser { return &AutoParser{} }

// Name implements Parser.
func (*AutoParser) Name() string { return "auto" }

// CanParse implements Parser: auto accepts everything.
func (*AutoParser) CanParse(string) bool { return true }

// Parse implements Parser. While the format is undecided, lines are buffered.
// Once sniffLimit lines have been seen the format is locked in and buffered
// lines are parsed with the chosen parser.
func (a *AutoParser) Parse(line string) []*model.LogEvent {
	if a.active != nil {
		return a.active.Parse(line)
	}
	a.pending = append(a.pending, line)
	if len(a.pending) >= sniffLimit {
		a.activate()
		out := a.output
		a.output = nil
		return out
	}
	return nil
}

// Flush implements Parser: forces format detection and drains all buffers.
func (a *AutoParser) Flush() []*model.LogEvent {
	if a.active == nil {
		a.activate()
	}
	out := a.output
	a.output = nil
	out = append(out, a.active.Flush()...)
	return out
}

// Issues implements Parser: delegates to the detected format.
func (a *AutoParser) Issues() int {
	if a.active == nil {
		return 0
	}
	return a.active.Issues()
}

// Detected returns the locked-in format name, or "" while undecided.
func (a *AutoParser) Detected() string {
	if a.active == nil {
		return ""
	}
	return a.active.Name()
}

// Reset implements Parser.
func (a *AutoParser) Reset() {
	a.active = nil
	a.pending = nil
	a.output = nil
}

// activate locks in the detected format and parses all pending lines.
func (a *AutoParser) activate() {
	a.active = detectFormat(a.pending)
	var out []*model.LogEvent
	for _, line := range a.pending {
		out = append(out, a.active.Parse(line)...)
	}
	a.pending = nil
	a.output = out
}

// detectFormat classifies the buffered lines into a concrete parser.
func detectFormat(lines []string) Parser {
	hasJSON := false
	hasJava := false
	hasNginx := false

	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if s[0] == '{' || s[0] == '[' {
			var v any
			if json.Unmarshal([]byte(s), &v) == nil {
				hasJSON = true
				continue
			}
		}
		if accessLogRe.MatchString(line) || errorLogRe.MatchString(line) {
			hasNginx = true
			continue
		}
		if isJavaEventStart(line) || isStackContinuation(line) || isExceptionLine(line) {
			hasJava = true
			continue
		}
	}

	switch {
	case hasJSON:
		return NewJSONParser()
	case hasJava:
		return NewJavaParser()
	case hasNginx:
		return NewNginxParser()
	default:
		return NewPlainTextParser()
	}
}
