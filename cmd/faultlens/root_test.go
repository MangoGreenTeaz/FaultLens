package main

import (
	"bytes"
	"strings"
	"testing"
)

// execute runs the root command with the given arguments and captures
// all output written to stdout and stderr.
func execute(args ...string) (string, error) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
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
	for _, want := range []string{"FaultLens", "version", "faultlens"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q; got:\n%s", want, out)
		}
	}
}

func TestNoArgsShowsHelp(t *testing.T) {
	out, err := execute()
	if err != nil {
		t.Fatalf("running with no args failed: %v", err)
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("expected usage to be shown with no args; got:\n%s", out)
	}
}

func TestUnknownCommandReturnsError(t *testing.T) {
	_, err := execute("does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown command, got nil")
	}
}
