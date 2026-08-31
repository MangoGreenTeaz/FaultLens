package normalize

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ip with port", "Connection refused 10.0.0.1:3306", "Connection refused <IP>:<PORT>"},
		{"ip without port", "connection to 192.168.1.5 timed out", "connection to <IP> timed out"},
		{"ipv6", "no route to host 2001:db8::1", "no route to host <IP>"},
		{"ipv6 with zone", "connect fe80::1ff:fe23:4567:890a failed", "connect <IP> failed"},
		{"uuid", "request 550e8400-e29b-41d4-a716-446655440000 failed", "request <UUID> failed"},
		{"number", "User 12345 login failed", "User <NUMBER> login failed"},
		{"numbers multiple", "attempt 3 of 10", "attempt <NUMBER> of <NUMBER>"},
		{"timestamp space", "2026-08-31 14:32:01 request timed out", "<TIME> request timed out"},
		{"timestamp rfc3339", "deadline 2026-08-31T14:32:01Z exceeded", "deadline <TIME> exceeded"},
		{"timestamp with offset", "at 2026-08-31T14:32:01.123+08:00", "at <TIME>"},
		{"url", "failed to connect to http://example.com:8080/api", "failed to connect to <URL>"},
		{"unix path", "/var/log/app.log not found", "<PATH> not found"},
		{"unix path deep", "open /usr/local/lib/libfoo.so denied", "open <PATH> denied"},
		{"windows path", "C:\\Program Files\\app\\config.yaml missing", "<PATH> missing"},
		{"hex id", "request 550e8400e29b41d4 failed", "request <HEX> failed"},
		{"no dynamic values", "database connection failed", "database connection failed"},
		{"http status protected", "HTTP 500 Internal Server Error", "HTTP 500 Internal Server Error"},
		{"http status short", "HTTP 404", "HTTP 404"},
		{"http version status", "got HTTP/1.1 502 response", "got HTTP 502 response"},
		{"status key protected", "status=503 upstream down", "status 503 upstream down"},
		{"empty message", "", ""},
		{"wall clock normalized", "timeout after 14:32:01", "timeout after <TIME>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := New()
			if got := n.Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeCollapsesDynamicValues(t *testing.T) {
	// Different dynamic values must produce the identical normalized shape.
	n := New()
	inputs := []string{
		"Connection refused 10.0.0.1:3306",
		"Connection refused 10.0.0.2:3306",
		"Connection refused 192.168.1.100:5432",
	}
	first := n.Normalize(inputs[0])
	for _, in := range inputs[1:] {
		if got := n.Normalize(in); got != first {
			t.Errorf("Normalize(%q) = %q, want %q (all must collapse)", in, got, first)
		}
	}
	if first != "Connection refused <IP>:<PORT>" {
		t.Errorf("collapsed shape = %q, want %q", first, "Connection refused <IP>:<PORT>")
	}
}

func TestNormalizeDistinctShapes(t *testing.T) {
	// Distinct error shapes must remain distinct (no over-collapsing).
	n := New()
	a := n.Normalize("Connection refused 10.0.0.1:3306")
	b := n.Normalize("HTTP 500")
	c := n.Normalize("User 12345 login failed")
	if a == b || a == c || b == c {
		t.Errorf("distinct errors collapsed: a=%q b=%q c=%q", a, b, c)
	}
}

func TestNormalizeStatusCodesStayDistinct(t *testing.T) {
	// 500 vs 404 must not collapse into the same normalized message.
	n := New()
	got500 := n.Normalize("HTTP 500 error")
	got404 := n.Normalize("HTTP 404 error")
	if got500 == got404 {
		t.Errorf("HTTP 500 and HTTP 404 collapsed into %q", got500)
	}
	if got500 != "HTTP 500 error" || got404 != "HTTP 404 error" {
		t.Errorf("status codes not preserved: %q / %q", got500, got404)
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	// Normalizing an already-normalized message must be a no-op.
	n := New()
	in := "Connection refused 10.0.0.1:3306 failed for request 550e8400-e29b-41d4-a716-446655440000"
	once := n.Normalize(in)
	twice := n.Normalize(once)
	if once != twice {
		t.Errorf("normalization not idempotent: %q != %q", once, twice)
	}
}
