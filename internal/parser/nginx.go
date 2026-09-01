package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// Nginx access log line:
//
//	127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 500 123
var accessLogRe = regexp.MustCompile(`^(\S+) (\S+) (\S+) \[([^\]]+)\] "(\S+) (\S+) ([^"]*)" (\d{3}) (-|\d+)(?: "([^"]*)")?(?: "([^"]*)")?$`)

// Nginx error log line:
//
//	2026/08/31 14:32:01 [error] 1234#0: *1 connect() failed (111: ...)
var errorLogRe = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)\] (\S+): (.*)$`)

// nginxErrorLevels maps Nginx error log severity tokens to canonical levels.
var nginxErrorLevels = map[string]model.LogLevel{
	"debug":  model.LevelDebug,
	"info":   model.LevelInfo,
	"notice": model.LevelInfo,
	"warn":   model.LevelWarn,
	"error":  model.LevelError,
	"crit":   model.LevelFatal,
	"alert":  model.LevelFatal,
	"emerg":  model.LevelFatal,
}

// nginxTimeLayout is the Nginx access log timestamp format.
const nginxTimeLayout = "02/Jan/2006:15:04:05 -0700"

// NginxParser parses Nginx access and error log lines. Access log fields are
// stored in LogEvent.Fields; error log severity is mapped onto LogEvent.Level.
type NginxParser struct {
	issues int
}

// NewNginxParser returns an NginxParser.
func NewNginxParser() *NginxParser { return &NginxParser{} }

// Name implements Parser.
func (*NginxParser) Name() string { return "nginx" }

// CanParse implements Parser: an access log or error log line.
func (*NginxParser) CanParse(line string) bool {
	return accessLogRe.MatchString(line) || errorLogRe.MatchString(line)
}

// Parse implements Parser.
func (p *NginxParser) Parse(line string) []*model.LogEvent {
	if m := accessLogRe.FindStringSubmatch(line); m != nil {
		return []*model.LogEvent{parseAccessLog(line, m)}
	}
	if m := errorLogRe.FindStringSubmatch(line); m != nil {
		return []*model.LogEvent{parseErrorLog(line, m)}
	}
	// Not an Nginx line: emit nothing, the caller counts it as a warning.
	p.issues++
	return nil
}

// Flush implements Parser. Nginx parsing is stateless.
func (*NginxParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (p *NginxParser) Issues() int { return p.issues }

// Reset implements Parser.
func (p *NginxParser) Reset() { p.issues = 0 }

// parseAccessLog builds a LogEvent from a matched access log line.
func parseAccessLog(raw string, m []string) *model.LogEvent {
	ev := &model.LogEvent{Raw: raw, Level: model.LevelUnknown, Message: raw}

	if t, err := time.Parse(nginxTimeLayout, m[4]); err == nil {
		ev.Timestamp = t
	}

	ev.Fields = map[string]string{
		"client_ip":   m[1],
		"remote_user": m[3],
		"method":      m[5],
		"path":        m[6],
		"protocol":    m[7],
		"status":      m[8],
		"size":        m[9],
	}
	if m[10] != "" {
		ev.Fields["referer"] = m[10]
	}
	if m[11] != "" {
		ev.Fields["user_agent"] = m[11]
	}

	// A 4xx/5xx status makes this an error-ish event for diagnostics.
	switch {
	case strings.HasPrefix(m[8], "5"):
		ev.Level = model.LevelError
	case strings.HasPrefix(m[8], "4"):
		ev.Level = model.LevelWarn
	default:
		ev.Level = model.LevelInfo
	}

	// Message is a compact summary of the request line.
	ev.Message = m[5] + " " + m[6] + " -> " + m[8]
	return ev
}

// parseErrorLog builds a LogEvent from a matched Nginx error log line.
func parseErrorLog(raw string, m []string) *model.LogEvent {
	ev := &model.LogEvent{Raw: raw, Message: m[4]}

	if t, err := time.Parse("2006/01/02 15:04:05", m[1]); err == nil {
		ev.Timestamp = t
	}
	ev.Level = nginxErrorLevels[strings.ToLower(m[2])]
	if ev.Level == "" {
		ev.Level = model.LevelUnknown
	}

	ev.Fields = map[string]string{
		"pid":  m[3],
		"type": "error_log",
	}
	return ev
}
