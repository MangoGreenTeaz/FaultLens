package parser

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// Field aliases accepted by the JSON parser.
var (
	timeKeys  = []string{"timestamp", "time", "ts"}
	levelKeys = []string{"level", "severity"}
	msgKeys   = []string{"message", "msg"}
	svcKeys   = []string{"service", "service_name"}
)

// JSONParser parses JSONL log lines into model.LogEvent values.
//
// Known fields are mapped onto the structured fields; anything else is kept
// verbatim in LogEvent.Fields. Malformed JSON lines produce no event so the
// caller can count them as parsing warnings.
type JSONParser struct {
	issues int
}

// NewJSONParser returns a JSONParser.
func NewJSONParser() *JSONParser { return &JSONParser{} }

// Name implements Parser.
func (*JSONParser) Name() string { return "json" }

// CanParse implements Parser: a line that starts with an object or array.
func (*JSONParser) CanParse(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}
	return s[0] == '{' || s[0] == '['
}

// Parse implements Parser.
func (p *JSONParser) Parse(line string) []*model.LogEvent {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		// Malformed JSON: emit nothing; the caller counts the warning.
		p.issues++
		return nil
	}

	ev := &model.LogEvent{Raw: line, Level: model.LevelUnknown, Message: line}
	ev.Timestamp = jsonTime(raw)
	if v := jsonString(raw, levelKeys); v != "" {
		ev.Level = model.ParseLevel(v)
	}
	if v := jsonString(raw, msgKeys); v != "" {
		ev.Message = v
	}
	ev.Service = jsonString(raw, svcKeys)
	ev.Fields = jsonFields(raw)

	return []*model.LogEvent{ev}
}

// Flush implements Parser. JSON parsing is stateless.
func (*JSONParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (p *JSONParser) Issues() int { return p.issues }

// Reset implements Parser.
func (p *JSONParser) Reset() { p.issues = 0 }

// jsonTime extracts the event timestamp from the recognized alias keys.
// Unparseable timestamps yield the zero time instead of failing.
func jsonTime(raw map[string]any) time.Time {
	for _, k := range timeKeys {
		s, ok := raw[k].(string)
		if !ok {
			continue
		}
		for _, layout := range tTimeLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
		// Fall back to space-separated shapes as well.
		for _, layout := range spaceTimeLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// jsonString returns the first recognized key's string value.
func jsonString(raw map[string]any, keys []string) string {
	for _, k := range keys {
		if s, ok := raw[k].(string); ok {
			return s
		}
	}
	return ""
}

// jsonFields copies every field that is not a recognized alias into the
// event's Fields map, converting scalar values to strings.
func jsonFields(raw map[string]any) map[string]string {
	fields := make(map[string]string)
	for k, v := range raw {
		if isKnownKey(k) {
			continue
		}
		switch val := v.(type) {
		case string:
			fields[k] = val
		case nil:
			fields[k] = ""
		default:
			if b, err := json.Marshal(val); err == nil {
				fields[k] = string(b)
			}
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// isKnownKey reports whether k is a recognized structured field alias.
func isKnownKey(k string) bool {
	for _, keys := range [][]string{timeKeys, levelKeys, msgKeys, svcKeys} {
		for _, key := range keys {
			if k == key {
				return true
			}
		}
	}
	return false
}
