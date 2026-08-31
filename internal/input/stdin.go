package input

import "os"

// NewStdinReader returns a Reader that streams log lines from standard input.
//
// It is used for piped usage such as:
//
//	cat app.log | faultlens
func NewStdinReader() *Reader {
	return NewReader(os.Stdin, "stdin")
}
