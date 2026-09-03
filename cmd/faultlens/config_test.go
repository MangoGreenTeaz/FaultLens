package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigCommands runs config init/show/validate in an isolated directory.
func TestConfigCommands(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old) //nolint:errcheck

	out, err := execute("config", "init")
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}
	if !strings.Contains(out, "wrote .faultlens.yaml") {
		t.Errorf("init output = %q", out)
	}
	if _, err := os.Stat(".faultlens.yaml"); err != nil {
		t.Fatalf(".faultlens.yaml not created: %v", err)
	}

	// A second init must refuse to overwrite.
	if _, err := execute("config", "init"); err == nil {
		t.Fatal("expected error when .faultlens.yaml already exists")
	}

	out, err = execute("config", "show")
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}
	for _, want := range []string{"min_baseline", "database_unavailable", "custom_rules", "terminal"} {
		if !strings.Contains(out, want) {
			t.Errorf("config show missing %q", want)
		}
	}

	out, err = execute("config", "validate")
	if err != nil {
		t.Fatalf("config validate failed: %v", err)
	}
	if !strings.Contains(out, "configuration is valid") {
		t.Errorf("validate output = %q", out)
	}
}

// TestConfigValidateInvalidFile checks that --config with bad values fails.
func TestConfigValidateInvalidFile(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.yaml")
	content := "anomaly:\n  min_baseline: 0\ncustom_rules:\n  - id: r1\n    severity: high\n"
	if err := os.WriteFile(bad, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("config", "validate", "--config", bad); err == nil {
		t.Fatal("expected validation error for invalid config")
	}
}

// TestConfigValidateMissingFile checks --config pointing to a missing file.
func TestConfigValidateMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := execute("config", "validate", "--config", path); err == nil {
		t.Fatal("expected error for missing --config file")
	}
}
