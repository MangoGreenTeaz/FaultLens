package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execute runs the root command with the given arguments, an empty stdin and
// captures all output written to stdout and stderr.
func execute(args ...string) (string, error) {
	return executeStdin("", args...)
}

// executeStdin runs the root command with the given stdin content.
func executeStdin(content string, args ...string) (string, error) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(content))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestVersionCommand(t *testing.T) {
	out, err := execute("version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.HasPrefix(out, "faultlens version ") {
		t.Errorf("expected output to start with %q, got %q", "faultlens version ", out)
	}
}

func TestHelpFlag(t *testing.T) {
	out, err := execute("--help")
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	for _, want := range []string{"FaultLens", "version", "errors", "timeline", "incident", "faultlens"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q; got:\n%s", want, out)
		}
	}
}

func TestNoArgsReadsEmptyStdin(t *testing.T) {
	// No file argument → stdin mode. Empty stdin must not error and should
	// still produce a (zero-event) report.
	out, err := execute()
	if err != nil {
		t.Fatalf("running with no args failed: %v", err)
	}
	if !strings.Contains(out, "Events:") {
		t.Errorf("expected summary output, got:\n%s", out)
	}
	if !strings.Contains(out, "Diagnosis") {
		t.Errorf("expected diagnosis section, got:\n%s", out)
	}
}

func TestUnknownCommandReturnsError(t *testing.T) {
	_, err := execute("does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown command, got nil")
	}
}

func TestAnalyzeStdin(t *testing.T) {
	logs := "2026-08-31 14:32:01 ERROR database connection failed\n"
	out, err := executeStdin(logs)
	if err != nil {
		t.Fatalf("stdin analysis failed: %v", err)
	}
	if !strings.Contains(out, "Events:       1") {
		t.Errorf("expected 1 event, got:\n%s", out)
	}
	if !strings.Contains(out, "Errors:       1") {
		t.Errorf("expected 1 error, got:\n%s", out)
	}
}

func TestAnalyzeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	if err := os.WriteFile(path, []byte("2026-08-31 14:32:01 ERROR boom\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := execute(path)
	if err != nil {
		t.Fatalf("file analysis failed: %v", err)
	}
	if !strings.Contains(out, "Events:       1") {
		t.Errorf("expected 1 event, got:\n%s", out)
	}
}

func TestAnalyzeMissingFile(t *testing.T) {
	_, err := execute(filepath.Join(t.TempDir(), "nope.log"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestIncidentCommand(t *testing.T) {
	// Strong database evidence chain: repeated db errors plus HTTP 5xx.
	var b strings.Builder
	for i := 0; i < 12; i++ {
		b.WriteString("2026-08-31 14:32:01 ERROR MySQL connection failed\n")
	}
	for i := 0; i < 12; i++ {
		b.WriteString("2026-08-31 14:32:05 ERROR HTTP 500 Internal Server Error\n")
	}
	out, err := executeStdin(b.String(), "incident")
	if err != nil {
		t.Fatalf("incident command failed: %v", err)
	}
	if !strings.Contains(out, "Root Cause:") {
		t.Errorf("expected Root Cause section, got:\n%s", out)
	}
	if !strings.Contains(out, "Database unavailable") {
		t.Errorf("expected database root cause, got:\n%s", out)
	}
}

func TestErrorsCommand(t *testing.T) {
	logs := "2026-08-31 14:32:01 ERROR boom\n2026-08-31 14:32:02 ERROR boom\n"
	out, err := executeStdin(logs, "errors")
	if err != nil {
		t.Fatalf("errors command failed: %v", err)
	}
	if !strings.Contains(out, "ERROR GROUPS") {
		t.Errorf("expected ERROR GROUPS section, got:\n%s", out)
	}
	if !strings.Contains(out, "Occurrences: 2") {
		t.Errorf("expected 2 occurrences, got:\n%s", out)
	}
}

func TestOutputJSONFlag(t *testing.T) {
	logs := "2026-08-31 14:32:01 ERROR boom\n"
	out, err := executeStdin(logs, "incident", "--output", "json")
	if err != nil {
		t.Fatalf("json output failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if _, ok := m["diagnosis"]; !ok {
		t.Error("JSON output missing diagnosis key")
	}
}

func TestOutputMarkdownFlag(t *testing.T) {
	logs := "2026-08-31 14:32:01 ERROR boom\n"
	out, err := executeStdin(logs, "incident", "--output", "markdown")
	if err != nil {
		t.Fatalf("markdown output failed: %v", err)
	}
	if !strings.Contains(out, "# Incident Report") {
		t.Errorf("expected markdown heading, got:\n%s", out)
	}
}

func TestOutputHTMLFlag(t *testing.T) {
	logs := "2026-08-31 14:32:01 ERROR boom\n"
	out, err := executeStdin(logs, "--output", "html")
	if err != nil {
		t.Fatalf("html output failed: %v", err)
	}
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("expected html document, got:\n%s", out)
	}
}

func TestOutputFileFlag(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")
	if _, err := executeStdin("2026-08-31 14:32:01 ERROR boom\n", "--output", "html", "-o", outPath); err != nil {
		t.Fatalf("file output failed: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("report file not written: %v", err)
	}
	if !strings.Contains(string(data), "FaultLens Report") {
		t.Error("report file has unexpected content")
	}
}

func TestInvalidFormatFlag(t *testing.T) {
	logs := "hello\n"
	if _, err := executeStdin(logs, "--format", "bogus"); err != nil {
		t.Fatalf("bogus format should fall back to auto, got error: %v", err)
	}
}

func TestInvalidFromFlag(t *testing.T) {
	_, err := executeStdin("", "--from", "not-a-time")
	if err == nil {
		t.Fatal("expected error for invalid --from value")
	}
}
