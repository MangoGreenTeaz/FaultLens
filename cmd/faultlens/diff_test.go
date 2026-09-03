package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiffCommand compares two real incident reports: redis-outage before,
// mysql-outage after. The diagnosis must change and error groups appear as
// added/removed.
func TestDiffCommand(t *testing.T) {
	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.json")
	afterPath := filepath.Join(dir, "after.json")

	beforeOut, err := execute("incident", filepath.Join("..", "..", "testdata", "incidents", "redis-outage.log"), "--output", "json")
	if err != nil {
		t.Fatalf("before report: %v", err)
	}
	afterOut, err := execute("incident", filepath.Join("..", "..", "testdata", "incidents", "mysql-outage.log"), "--output", "json")
	if err != nil {
		t.Fatalf("after report: %v", err)
	}
	if err := os.WriteFile(beforePath, []byte(beforeOut), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(afterPath, []byte(afterOut), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := execute("diff", beforePath, afterPath)
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}
	for _, want := range []string{"Error Groups", "Diagnosis changed:", "Redis unavailable", "Database unavailable"} {
		if !strings.Contains(out, want) {
			t.Errorf("diff output missing %q; got:\n%s", want, out)
		}
	}
}

// TestDiffCommandSame verifies identical reports yield "no differences".
func TestDiffCommandSame(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	out, err := execute("incident", filepath.Join("..", "..", "testdata", "incidents", "mysql-outage.log"), "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}

	diffOut, err := execute("diff", a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diffOut, "No differences detected") {
		t.Errorf("expected no differences, got:\n%s", diffOut)
	}
}

// TestDiffCommandMissingFile verifies the error path.
func TestDiffCommandMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := execute("diff", filepath.Join(dir, "nope.json"), filepath.Join(dir, "nope2.json"))
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}
