package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func collect(r *Reader) ([]string, error) {
	var lines []string
	for r.Scan() {
		lines = append(lines, r.Text())
	}
	return lines, r.Err()
}

func TestNewFileReaderReadsLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := NewFileReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if got := r.Name(); got != path {
		t.Errorf("Name() = %q, want %q", got, path)
	}

	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"line one", "line two", "line three"}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestNewFileReaderMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.log")
	if _, err := NewFileReader(path); err == nil {
		t.Fatalf("expected error for missing file %q, got nil", path)
	}
}

func TestReaderEmptyInput(t *testing.T) {
	r := NewReader(strings.NewReader(""), "test")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("empty input produced %d lines, want 0", len(lines))
	}
}

func TestReaderCRLF(t *testing.T) {
	r := NewReader(strings.NewReader("alpha\r\nbeta\r\n"), "test")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"alpha", "beta"}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestReaderNoTrailingNewline(t *testing.T) {
	// The last line has no newline; it must still be emitted.
	r := NewReader(strings.NewReader("first\nlast line without newline"), "test")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"first", "last line without newline"}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	if lines[1] != want[1] {
		t.Errorf("last line = %q, want %q", lines[1], want[1])
	}
}

func TestReaderVeryLongLineDoesNotAbort(t *testing.T) {
	// A line longer than the internal buffer must not stop the scan:
	// surrounding lines must still be read.
	long := strings.Repeat("x", maxLineSize+1000)
	r := NewReader(strings.NewReader("first\n"+long+"\nlast\n"), "test")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) < 3 {
		t.Fatalf("got %d lines, want at least 3 (over-long line must not abort)", len(lines))
	}
	if lines[0] != "first" {
		t.Errorf("first line = %q, want %q", lines[0], "first")
	}
	if lines[len(lines)-1] != "last" {
		t.Errorf("last line = %q, want %q", lines[len(lines)-1], "last")
	}
}

func TestReaderBlankLinesAreKept(t *testing.T) {
	r := NewReader(strings.NewReader("a\n\nb\n"), "test")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "", "b"}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	if lines[1] != "" {
		t.Errorf("blank line = %q, want empty string", lines[1])
	}
}
