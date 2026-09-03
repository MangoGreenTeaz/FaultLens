package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLog(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExpandPathsDirectory(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "a.log", "x")
	writeLog(t, dir, "b.log", "x")
	writeLog(t, dir, "notes.txt", "x") // non-log file must be ignored
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	writeLog(t, sub, "c.log", "x")

	files, err := expandPaths([]string{dir}, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(dir, "a.log"),
		filepath.Join(dir, "b.log"),
		filepath.Join(sub, "c.log"),
	}
	if len(files) != len(want) {
		t.Fatalf("got %d files %v, want %d", len(files), files, len(want))
	}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("files[%d] = %q, want %q", i, files[i], want[i])
		}
	}
}

func TestExpandPathsExclude(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "app.log", "x")
	writeLog(t, dir, "app.debug.log", "x")

	files, err := expandPaths([]string{dir}, "*.debug.log")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "app.log" {
		t.Errorf("exclude failed: %v", files)
	}
}

func TestExpandPathsGlob(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "a.log", "x")
	writeLog(t, dir, "b.log", "x")
	writeLog(t, dir, "c.txt", "x")

	files, err := expandPaths([]string{filepath.Join(dir, "*.log")}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("glob matched %v, want 2 files", files)
	}
}

func TestExpandPathsMultipleArgs(t *testing.T) {
	dir := t.TempDir()
	f1 := writeLog(t, dir, "a.log", "x")
	f2 := writeLog(t, dir, "b.log", "x")

	files, err := expandPaths([]string{f1, f2, f1}, "") // duplicates deduped
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("got %v, want 2 unique files", files)
	}
}

func TestExpandPathsNoMatch(t *testing.T) {
	dir := t.TempDir()
	files, err := expandPaths([]string{filepath.Join(dir, "*.txt")}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("got %v, want no matches", files)
	}
}

func TestExpandPathsMissing(t *testing.T) {
	if _, err := expandPaths([]string{filepath.Join(t.TempDir(), "nope.log")}, ""); err == nil {
		t.Fatal("expected error for missing path")
	}
}

// TestMultiFileCLI runs the real CLI against a directory of logs and verifies
// events from both files are merged into one analysis.
func TestMultiFileCLI(t *testing.T) {
	dir := t.TempDir()
	writeLog(t, dir, "a.log", "2026-08-31 14:32:01 ERROR boom\n2026-08-31 14:32:02 ERROR boom\n")
	writeLog(t, dir, "b.log", "2026-08-31 14:32:03 ERROR boom\n")

	out, err := execute(dir)
	if err != nil {
		t.Fatalf("directory analysis failed: %v", err)
	}
	if !strings.Contains(out, "Events:       3") {
		t.Errorf("expected 3 merged events, got:\n%s", out)
	}
	if !strings.Contains(out, "Errors:       3") {
		t.Errorf("expected 3 merged errors, got:\n%s", out)
	}
}

// TestMultiFileSourceTracking checks that error group examples carry their
// origin file via the JSON output.
func TestMultiFileSourceTracking(t *testing.T) {
	dir := t.TempDir()
	f1 := writeLog(t, dir, "a.log", "2026-08-31 14:32:01 ERROR boom\n")
	f2 := writeLog(t, dir, "b.log", "2026-08-31 14:32:02 ERROR boom\n")

	out, err := execute(f1, f2, "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.log") || !strings.Contains(out, "b.log") {
		t.Errorf("expected both source files in output, got:\n%s", out)
	}
}
