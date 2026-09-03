package parser

import (
	"encoding/json"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// DockerJSONParser parses Docker daemon JSON log lines:
//
//	{"log":"error: db down","stream":"stderr","time":"2026-08-31T14:32:01.123Z"}
type DockerJSONParser struct{}

// NewDockerJSONParser returns a DockerJSONParser.
func NewDockerJSONParser() *DockerJSONParser { return &DockerJSONParser{} }

// Name implements Parser.
func (*DockerJSONParser) Name() string { return "docker" }

// CanParse implements Parser: a JSON object carrying a "log" key plus a
// "stream" or "time" key.
func (*DockerJSONParser) CanParse(line string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return false
	}
	if _, ok := m["log"]; !ok {
		return false
	}
	_, hasStream := m["stream"]
	_, hasTime := m["time"]
	return hasStream || hasTime
}

// Parse implements Parser.
func (*DockerJSONParser) Parse(line string) []*model.LogEvent {
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return nil
	}
	log, _ := m["log"].(string)
	stream, _ := m["stream"].(string)

	ev := &model.LogEvent{
		Raw:     line,
		Message: log,
		Level:   model.LevelUnknown,
	}
	if t, ok := m["time"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			ev.Timestamp = parsed
		}
	}
	if stream != "" {
		ev.Fields = map[string]string{"stream": stream}
	}
	return []*model.LogEvent{ev}
}

// Flush implements Parser. Docker parsing is stateless.
func (*DockerJSONParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (*DockerJSONParser) Issues() int { return 0 }

// Reset implements Parser.
func (*DockerJSONParser) Reset() {}
