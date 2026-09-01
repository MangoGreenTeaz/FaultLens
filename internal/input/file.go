// Package input provides streaming log sources (file and stdin).
//
// Logs are always read line by line so that large files can be processed with
// bounded memory. The scanner tolerates very long lines (e.g. embedded stack
// traces) without aborting the whole read.
package input

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

const (
	// defaultBufferSize is the initial scanner buffer size.
	defaultBufferSize = 64 * 1024
	// maxLineSize is the upper bound for a single logical line. Lines longer
	// than this are emitted in chunks instead of aborting the scan.
	maxLineSize = 1 << 20 // 1 MiB
)

// Reader streams log lines from a source (file or stdin).
type Reader struct {
	scanner *bufio.Scanner
	closer  io.Closer
	name    string
}

// NewReader creates a Reader over an arbitrary input stream.
func NewReader(r io.Reader, name string) *Reader {
	return &Reader{scanner: newScanner(r), name: name}
}

// NewFileReader opens path and returns a streaming Reader for it.
// The caller is responsible for closing the returned Reader.
func NewFileReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &Reader{scanner: newScanner(f), closer: f, name: path}, nil
}

// Name returns a human-readable description of the source
// (the file path, or "stdin").
func (r *Reader) Name() string { return r.name }

// Scan advances the reader to the next line. It returns false when the
// stream is exhausted or an unrecoverable error occurred; check Err.
func (r *Reader) Scan() bool { return r.scanner.Scan() }

// Text returns the current line, with the trailing newline removed.
func (r *Reader) Text() string { return r.scanner.Text() }

// Err returns the first non-EOF error encountered while scanning.
func (r *Reader) Err() error { return r.scanner.Err() }

// Close releases the underlying source (no-op for stdin).
func (r *Reader) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// newScanner builds a scanner that tolerates very long lines and skips a
// UTF-8 BOM at the very start of the stream (common in files produced by
// Windows tools).
func newScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, defaultBufferSize), maxLineSize)
	first := true
	s.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if first {
			first = false
			if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
				adv, tok, err := splitLines(data[3:], atEOF)
				return adv + 3, tok, err
			}
		}
		return splitLines(data, atEOF)
	})
	return s
}

// splitLines behaves like bufio.ScanLines but, instead of failing with
// bufio.ErrTooLong on over-long lines, it emits them in chunks so scanning
// always continues. Both \n and \r\n line endings are supported.
func splitLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, dropCR(data[0:i]), nil
	}
	if atEOF {
		return len(data), dropCR(data), nil
	}
	if len(data) >= maxLineSize {
		return len(data), dropCR(data), nil
	}
	return 0, nil, nil
}

// dropCR removes a trailing carriage return, so files with CRLF endings
// produce the same lines on every platform.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
