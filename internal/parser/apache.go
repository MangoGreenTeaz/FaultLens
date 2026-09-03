package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// Apache access log formats: common, combined and vhost-combined. The vhost
// prefix (e.g. "example.com:80") distinguishes Apache rows from Nginx rows
// during auto detection.
var apacheRe = regexp.MustCompile(`^(?:(\S+:\d+)\s+)?(\S+) (\S+) (\S+) \[([^\]]+)\] "(\S+) (\S+) ([^"]*)" (\d{3}) (-|\d+)(?: "([^"]*)")?(?: "([^"]*)")?$`)

// ApacheParser parses Apache access log lines (common / combined / vhost).
type ApacheParser struct{}

// NewApacheParser returns an ApacheParser.
func NewApacheParser() *ApacheParser { return &ApacheParser{} }

// Name implements Parser.
func (*ApacheParser) Name() string { return "apache" }

// CanParse implements Parser.
func (*ApacheParser) CanParse(line string) bool {
	return apacheRe.MatchString(line)
}

// Parse implements Parser.
func (*ApacheParser) Parse(line string) []*model.LogEvent {
	m := apacheRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	return []*model.LogEvent{parseApacheLine(line, m)}
}

// Flush implements Parser. Apache parsing is stateless.
func (*ApacheParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (*ApacheParser) Issues() int { return 0 }

// Reset implements Parser.
func (*ApacheParser) Reset() {}

// parseApacheLine builds a LogEvent from a matched Apache access log line.
// Capture groups: 1=vhost, 2=ip, 3=ident, 4=auth, 5=time, 6=method, 7=path,
// 8=protocol, 9=status, 10=size, 11=referer, 12=ua.
func parseApacheLine(raw string, m []string) *model.LogEvent {
	ev := &model.LogEvent{Raw: raw, Level: model.LevelUnknown, Message: raw}

	if t, err := time.Parse(nginxTimeLayout, m[5]); err == nil {
		ev.Timestamp = t
	}

	ev.Fields = map[string]string{
		"client_ip":   m[2],
		"remote_user": m[4],
		"method":      m[6],
		"path":        m[7],
		"protocol":    m[8],
		"status":      m[9],
		"size":        m[10],
	}
	if m[1] != "" {
		ev.Fields["vhost"] = m[1]
	}
	if m[11] != "" {
		ev.Fields["referer"] = m[11]
	}
	if m[12] != "" {
		ev.Fields["user_agent"] = m[12]
	}

	switch {
	case strings.HasPrefix(m[9], "5"):
		ev.Level = model.LevelError
	case strings.HasPrefix(m[9], "4"):
		ev.Level = model.LevelWarn
	default:
		ev.Level = model.LevelInfo
	}

	ev.Message = m[6] + " " + m[7] + " -> " + m[9]
	return ev
}
