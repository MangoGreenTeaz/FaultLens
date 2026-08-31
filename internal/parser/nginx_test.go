package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

func TestNginxParseAccessLog(t *testing.T) {
	p := NewNginxParser()
	line := `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 500 123`
	events := feedAll(p, line)
	requireEventCount(t, events, 1)

	ev := events[0]
	want := time.Date(2026, 8, 31, 14, 32, 1, 0, time.FixedZone("", 8*3600))
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
	if ev.Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR (5xx)", ev.Level)
	}
	fields := ev.Fields
	for key, wantVal := range map[string]string{
		"client_ip": "127.0.0.1",
		"method":    "GET",
		"path":      "/api",
		"protocol":  "HTTP/1.1",
		"status":    "500",
		"size":      "123",
	} {
		if fields[key] != wantVal {
			t.Errorf("Fields[%s] = %q, want %q", key, fields[key], wantVal)
		}
	}
	if !strings.Contains(ev.Message, "GET /api") {
		t.Errorf("Message = %q, want request summary", ev.Message)
	}
}

func TestNginxParseAccessLog2xx(t *testing.T) {
	p := NewNginxParser()
	events := feedAll(p, `10.0.0.5 - admin [31/Aug/2026:10:00:00 +0000] "POST /users HTTP/1.1" 201 45`)
	requireEventCount(t, events, 1)
	if events[0].Level != model.LevelInfo {
		t.Errorf("Level = %q, want INFO (2xx)", events[0].Level)
	}
	if events[0].Fields["remote_user"] != "admin" {
		t.Errorf("remote_user = %q, want admin", events[0].Fields["remote_user"])
	}
	if events[0].Fields["status"] != "201" {
		t.Errorf("status = %q, want 201", events[0].Fields["status"])
	}
}

func TestNginxParseAccessLogWithRefererAndUA(t *testing.T) {
	p := NewNginxParser()
	line := `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET / HTTP/1.1" 200 612 "https://example.com/" "Mozilla/5.0"`
	events := feedAll(p, line)
	requireEventCount(t, events, 1)
	if events[0].Fields["referer"] != "https://example.com/" {
		t.Errorf("referer = %q", events[0].Fields["referer"])
	}
	if events[0].Fields["user_agent"] != "Mozilla/5.0" {
		t.Errorf("user_agent = %q", events[0].Fields["user_agent"])
	}
}

func TestNginxParseErrorLog(t *testing.T) {
	p := NewNginxParser()
	line := `2026/08/31 14:32:01 [error] 1234#0: *1 connect() failed (111: Connection refused) while connecting to upstream, client: 127.0.0.1, server: localhost, request: "GET /api HTTP/1.1", upstream: "http://127.0.0.1:3306"`
	events := feedAll(p, line)
	requireEventCount(t, events, 1)

	ev := events[0]
	want := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	if !ev.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, want)
	}
	if ev.Level != model.LevelError {
		t.Errorf("Level = %q, want ERROR", ev.Level)
	}
	if ev.Fields["pid"] != "1234#0" {
		t.Errorf("pid = %q, want 1234#0", ev.Fields["pid"])
	}
	if !strings.Contains(ev.Message, "connect() failed") {
		t.Errorf("Message = %q, want upstream failure detail", ev.Message)
	}
}

func TestNginxParseCritLevel(t *testing.T) {
	p := NewNginxParser()
	events := feedAll(p, `2026/08/31 14:32:01 [crit] 999#0: *2 open() "/var/www/index.html" failed (13: Permission denied)`)
	requireEventCount(t, events, 1)
	if events[0].Level != model.LevelFatal {
		t.Errorf("Level = %q, want FATAL (crit)", events[0].Level)
	}
}

func TestNginxParseNonNginxLine(t *testing.T) {
	p := NewNginxParser()
	events := feedAll(p, "2026-08-31 14:32:01 ERROR some other format")
	requireEventCount(t, events, 0)
}

func TestNginxCanParse(t *testing.T) {
	p := NewNginxParser()
	if !p.CanParse(`127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET / HTTP/1.1" 200 1`) {
		t.Error("CanParse should accept access log")
	}
	if !p.CanParse(`2026/08/31 14:32:01 [error] 1#0: *1 connect() failed`) {
		t.Error("CanParse should accept error log")
	}
	if p.CanParse("2026-08-31 14:32:01 ERROR x") {
		t.Error("CanParse should reject unrelated formats")
	}
}
