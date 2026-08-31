package input

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNewStdinReader(t *testing.T) {
	r := NewStdinReader()
	if r == nil {
		t.Fatal("NewStdinReader() returned nil")
	}
	if got := r.Name(); got != "stdin" {
		t.Errorf("Name() = %q, want %q", got, "stdin")
	}
}

func TestNewReaderOverBuffer(t *testing.T) {
	// NewReader must work with any io.Reader, e.g. a bytes.Buffer.
	r := NewReader(bytes.NewBufferString("x\n"), "buffer")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 1 || lines[0] != "x" {
		t.Errorf("got lines %v, want [x]", lines)
	}
}

func TestNewReaderFromDiscard(t *testing.T) {
	// Reading from an empty stream is fine and yields no lines.
	r := NewReader(io.LimitReader(strings.NewReader(""), 0), "empty")
	lines, err := collect(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("got %d lines, want 0", len(lines))
	}
}
