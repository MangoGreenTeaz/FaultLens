package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIIncidentFixture runs the real CLI against the MySQL outage fixture.
func TestCLIIncidentFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "incidents", "mysql-outage.log")
	out, err := execute("incident", path)
	if err != nil {
		t.Fatalf("incident command failed: %v", err)
	}
	for _, want := range []string{"Incident detected", "Root Cause:", "Database unavailable", "Evidence:", "Recommended:"} {
		if !strings.Contains(out, want) {
			t.Errorf("incident output missing %q", want)
		}
	}
}

// TestCLIErrorsFixture runs the errors view against the Redis outage fixture.
func TestCLIErrorsFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "incidents", "redis-outage.log")
	out, err := execute("errors", path)
	if err != nil {
		t.Fatalf("errors command failed: %v", err)
	}
	if !strings.Contains(out, "ERROR GROUPS") {
		t.Error("expected ERROR GROUPS section")
	}
	if !strings.Contains(out, "redis timeout") {
		t.Errorf("expected the redis error group, got:\n%s", out)
	}
}

// TestCLIJSONFixture verifies the JSON output against the OOM fixture.
func TestCLIJSONFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "incidents", "oom.log")
	out, err := execute("incident", path, "--output", "json")
	if err != nil {
		t.Fatalf("json output failed: %v", err)
	}
	for _, want := range []string{`"root_cause"`, "Out of memory", `"severity": "critical"`} {
		if !strings.Contains(out, want) {
			t.Errorf("json output missing %q", want)
		}
	}
}

// TestCLIMarkdownFixture verifies markdown output against the HTTP 5xx
// fixture (which must report insufficient evidence).
func TestCLIMarkdownFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "incidents", "http-5xx.log")
	out, err := execute("incident", path, "--output", "markdown")
	if err != nil {
		t.Fatalf("markdown output failed: %v", err)
	}
	if !strings.Contains(out, "Insufficient evidence") {
		t.Errorf("5xx-only fixture must report insufficient evidence, got:\n%s", out)
	}
}
