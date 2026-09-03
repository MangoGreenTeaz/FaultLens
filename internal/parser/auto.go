package parser

import (
	"encoding/json"
	"strings"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
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
	hasDocker := false
	hasJSON := false
	hasJava := false
	hasSyslog := false
	hasNginx := false
	hasApache := false
	hasK8s := false
	hasPython := false

	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if dockerJSONCanParse(s) {
			hasDocker = true
			continue
		}
		if s[0] == '{' || s[0] == '[' {
			var v any
			if json.Unmarshal([]byte(s), &v) == nil {
				hasJSON = true
				continue
			}
		}
		// Kubernetes rows also match the Python pattern structurally
		// ("stderr F error: ..."), so they must be checked first.
		if k8sRe.MatchString(line) {
			hasK8s = true
			continue
		}
		// Python rows use comma-fractional timestamps, which Go's time.Parse
		// accepts, so they would be misclassified as Java. Check Python
		// before Java.
		if pythonRe.MatchString(line) {
			hasPython = true
			continue
		}
		if isJavaEventStart(line) || isStackContinuation(line) || isExceptionLine(line) {
			hasJava = true
			continue
		}
		if syslog3164Re.MatchString(line) || syslog5424Re.MatchString(line) {
			hasSyslog = true
			continue
		}
		if accessLogRe.MatchString(line) || errorLogRe.MatchString(line) {
			hasNginx = true
			continue
		}
		if apacheRe.MatchString(line) {
			hasApache = true
			continue
		}
	}

	switch {
	// Docker JSON is a specific subset of general JSON: check it first so
	// container logs get their dedicated parser.
	case hasDocker:
		return NewDockerJSONParser()
	case hasJSON:
		return NewJSONParser()
	case hasJava:
		return NewJavaParser()
	case hasSyslog:
		return NewSyslogParser()
	case hasNginx:
		return NewNginxParser()
	case hasApache:
		return NewApacheParser()
	case hasPython:
		return NewPythonParser()
	case hasK8s:
		return NewKubernetesParser()
	default:
		return NewPlainTextParser()
	}
}

// dockerJSONCanParse reports whether s is a Docker JSON log object (a "log"
// key plus a "stream" or "time" key).
func dockerJSONCanParse(s string) bool {
	if len(s) == 0 || s[0] != '{' {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return false
	}
	if _, ok := m["log"]; !ok {
		return false
	}
	_, hasStream := m["stream"]
	_, hasTime := m["time"]
	return hasStream || hasTime
}
