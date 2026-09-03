package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// Syslog RFC 3164: <134>Aug 31 14:32:01 hostname app[123]: error message
var syslog3164Re = regexp.MustCompile(`^<(\d+)>([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+([^:]+):\s*(.*)$`)

// Syslog RFC 5424: <134>1 2026-08-31T14:32:01.123Z hostname app 123 ID - error message
var syslog5424Re = regexp.MustCompile(`^<(\d+)>\d+\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.*)$`)

// syslogLevels maps the syslog severity number (PRI % 8) to a LogLevel.
var syslogLevels = map[int]model.LogLevel{
	0: model.LevelFatal, // emerg
	1: model.LevelFatal, // alert
	2: model.LevelFatal, // crit
	3: model.LevelError, // err
	4: model.LevelWarn,  // warning
	5: model.LevelInfo,  // notice
	6: model.LevelInfo,  // info
	7: model.LevelDebug, // debug
}

// syslogTimeLayout is the RFC 3164 timestamp (no year, so the current year
// is assumed).
const syslogTimeLayout = "Jan _2 15:04:05"

// SyslogParser parses RFC 3164 and RFC 5424 syslog lines into a single
// module, per the V2 plan.
type SyslogParser struct{}

// NewSyslogParser returns a SyslogParser.
func NewSyslogParser() *SyslogParser { return &SyslogParser{} }

// Name implements Parser.
func (*SyslogParser) Name() string { return "syslog" }

// CanParse implements Parser.
func (*SyslogParser) CanParse(line string) bool {
	return syslog3164Re.MatchString(line) || syslog5424Re.MatchString(line)
}

// Parse implements Parser.
func (*SyslogParser) Parse(line string) []*model.LogEvent {
	if m := syslog5424Re.FindStringSubmatch(line); m != nil {
		return []*model.LogEvent{parseSyslog5424(line, m)}
	}
	if m := syslog3164Re.FindStringSubmatch(line); m != nil {
		return []*model.LogEvent{parseSyslog3164(line, m)}
	}
	return nil
}

// Flush implements Parser. Syslog parsing is stateless.
func (*SyslogParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (*SyslogParser) Issues() int { return 0 }

// Reset implements Parser.
func (*SyslogParser) Reset() {}

// parseSyslog3164 builds a LogEvent from a matched RFC 3164 line.
// Groups: 1=pri, 2=time, 3=hostname, 4=tag, 5=message.
func parseSyslog3164(raw string, m []string) *model.LogEvent {
	ev := &model.LogEvent{Raw: raw, Message: m[5]}

	ev.Level = syslogLevel(pri(m[1]))
	if t, err := time.ParseInLocation(syslogTimeLayout, m[2], time.Local); err == nil {
		// RFC 3164 carries no year; assume the current year.
		ev.Timestamp = t.AddDate(time.Now().Year()-t.Year(), 0, 0)
	}

	// tag is either "app" or "app[123]".
	tag := m[4]
	ev.Service = tag
	if i := strings.IndexByte(tag, '['); i >= 0 {
		ev.Service = tag[:i]
		if j := strings.IndexByte(tag, ']'); j > i {
			ev.Fields = map[string]string{"pid": tag[i+1 : j]}
		}
	}

	return ev
}

// parseSyslog5424 builds a LogEvent from a matched RFC 5424 line.
// Groups: 1=pri, 2=timestamp, 3=hostname, 4=appname, 5=procid, 6=msgid,
// 7=structured-data, 8=message.
func parseSyslog5424(raw string, m []string) *model.LogEvent {
	ev := &model.LogEvent{Raw: raw, Message: m[8], Service: m[4]}

	ev.Level = syslogLevel(pri(m[1]))
	if t, err := time.Parse(time.RFC3339Nano, m[2]); err == nil {
		ev.Timestamp = t
	}

	ev.Fields = map[string]string{
		"hostname": m[3],
		"appname":  m[4],
		"procid":   m[5],
		"msgid":    m[6],
		"sd":       m[7],
	}
	return ev
}

// syslogLevel converts a PRI value into the canonical LogLevel.
func syslogLevel(pri int) model.LogLevel {
	if lvl, ok := syslogLevels[pri%8]; ok {
		return lvl
	}
	return model.LevelUnknown
}

// pri parses the numeric part of a "<134>" PRI token.
func pri(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}
