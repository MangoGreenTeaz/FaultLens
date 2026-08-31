package parser

import (
	"strings"
	"testing"

	"github.com/faultlens/faultlens/internal/model"
)

func TestAutoDetectJSON(t *testing.T) {
	p := NewAutoParser()
	lines := []string{
		`{"timestamp":"2026-08-31T14:32:01Z","level":"INFO","message":"started","service":"api"}`,
		`{"timestamp":"2026-08-31T14:32:02Z","level":"ERROR","message":"db down","service":"api"}`,
	}
	var all []*model.LogEvent
	for i := 0; i < sniffLimit; i++ {
		all = append(all, p.Parse(lines[i%len(lines)])...)
	}
	if len(all) == 0 {
		t.Fatal("expected events from JSON detection")
	}
	if p.active == nil || p.active.Name() != "json" {
		t.Errorf("detected format = %v, want json", p.active.Name())
	}
	if all[0].Level != model.LevelInfo {
		t.Errorf("first event Level = %q, want INFO", all[0].Level)
	}
	if all[0].Service != "api" {
		t.Errorf("first event Service = %q, want api", all[0].Service)
	}
}

func TestAutoDetectJavaWithStackTrace(t *testing.T) {
	p := NewAutoParser()
	lines := []string{
		"2026-08-31 14:32:01 ERROR Database query failed",
		"java.sql.SQLException: Connection refused",
		"    at com.example.UserService.query(UserService.java:182)",
	}
	var all []*model.LogEvent
	for i := 0; i < sniffLimit; i++ {
		all = append(all, p.Parse(lines[i%len(lines)])...)
	}
	all = append(all, p.Flush()...)
	if len(all) == 0 {
		t.Fatal("expected events from Java detection")
	}
	if p.active == nil || p.active.Name() != "java" {
		t.Errorf("detected format = %v, want java", p.active.Name())
	}
	if !strings.Contains(all[0].StackTrace, "SQLException") {
		t.Errorf("Java events should aggregate the stack trace")
	}
}

func TestAutoDetectNginx(t *testing.T) {
	p := NewAutoParser()
	line := `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 500 123`
	var all []*model.LogEvent
	for i := 0; i < sniffLimit; i++ {
		all = append(all, p.Parse(line)...)
	}
	if len(all) == 0 {
		t.Fatal("expected events from Nginx detection")
	}
	if p.active == nil || p.active.Name() != "nginx" {
		t.Errorf("detected format = %v, want nginx", p.active.Name())
	}
	if all[0].Fields["status"] != "500" {
		t.Errorf("status = %q, want 500", all[0].Fields["status"])
	}
}

func TestAutoDetectPlain(t *testing.T) {
	p := NewAutoParser()
	// A timestamped line without a severity level has no Java signature,
	// so it must be detected as plain text.
	line := "2026-08-31 14:32:01 user logged in"
	var all []*model.LogEvent
	for i := 0; i < sniffLimit; i++ {
		all = append(all, p.Parse(line)...)
	}
	if len(all) == 0 {
		t.Fatal("expected events from plain detection")
	}
	if p.active == nil || p.active.Name() != "plain" {
		t.Errorf("detected format = %v, want plain", p.active.Name())
	}
}

func TestAutoBuffersUntilLimit(t *testing.T) {
	p := NewAutoParser()
	line := "2026-08-31 14:32:01 user logged in"
	// Before sniffLimit lines no events are produced.
	for i := 0; i < sniffLimit-1; i++ {
		if out := p.Parse(line); len(out) != 0 {
			t.Fatalf("expected buffering, got %d events before limit", len(out))
		}
	}
	// The line that reaches the limit triggers parsing of all buffered lines.
	out := p.Parse(line)
	if len(out) != sniffLimit {
		t.Fatalf("expected %d buffered events, got %d", sniffLimit, len(out))
	}
	if p.active == nil || p.active.Name() != "plain" {
		t.Errorf("detected format = %v, want plain", p.active.Name())
	}
}

func TestAutoEmptyInput(t *testing.T) {
	p := NewAutoParser()
	events := p.Flush()
	if len(events) != 0 {
		t.Errorf("empty input produced %d events, want 0", len(events))
	}
	if p.active == nil {
		t.Fatal("Flush should still lock in a format (plain fallback)")
	}
	if p.active.Name() != "plain" {
		t.Errorf("empty input detected %v, want plain", p.active.Name())
	}
}

func TestAutoBlankLinesSkipped(t *testing.T) {
	p := NewAutoParser()
	for i := 0; i < sniffLimit; i++ {
		_ = p.Parse("")
	}
	events := p.Flush()
	if len(events) != 0 {
		t.Errorf("blank input produced %d events, want 0", len(events))
	}
}
