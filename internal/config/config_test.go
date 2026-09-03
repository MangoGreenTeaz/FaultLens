package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	d := Default()
	if d.Anomaly.MinBaseline != 5 {
		t.Errorf("MinBaseline = %d, want 5", d.Anomaly.MinBaseline)
	}
	if d.Anomaly.ZScore != 3.0 {
		t.Errorf("ZScore = %v, want 3.0", d.Anomaly.ZScore)
	}
	if len(d.Rules) != 6 {
		t.Errorf("Rules = %d, want 6", len(d.Rules))
	}
	if d.Output.Format != "terminal" {
		t.Errorf("Output.Format = %q, want terminal", d.Output.Format)
	}
	if d.Rules["database_unavailable"].Enabled == nil || !*d.Rules["database_unavailable"].Enabled {
		t.Error("database rule should be enabled by default")
	}
	if len(d.Rules["database_unavailable"].StrongKeywords) == 0 {
		t.Error("database rule should have default strong keywords")
	}
}

func TestMergeAnomalyOverride(t *testing.T) {
	dst := Default()
	src := Config{Anomaly: AnomalyConfig{MinErrors: 50}}
	merge(dst, &src)
	if dst.Anomaly.MinErrors != 50 {
		t.Errorf("MinErrors = %d, want 50", dst.Anomaly.MinErrors)
	}
	if dst.Anomaly.MinBaseline != 5 {
		t.Errorf("MinBaseline changed to %d, want 5 (zero value must not overwrite)", dst.Anomaly.MinBaseline)
	}
}

func TestMergeRulePartialOverride(t *testing.T) {
	dst := Default()
	src := Config{Rules: map[string]RuleConfig{
		"database_unavailable": {Threshold: 30},
	}}
	merge(dst, &src)
	got := dst.Rules["database_unavailable"]
	if got.Threshold != 30 {
		t.Errorf("Threshold = %d, want 30", got.Threshold)
	}
	if len(got.StrongKeywords) == 0 {
		t.Error("StrongKeywords lost during partial merge")
	}
}

func TestMergeRuleDisable(t *testing.T) {
	dst := Default()
	f := false
	src := Config{Rules: map[string]RuleConfig{
		"redis_unavailable": {Enabled: &f},
	}}
	merge(dst, &src)
	if *dst.Rules["redis_unavailable"].Enabled {
		t.Error("redis rule should be disabled after merge")
	}
}

func TestMergeRuleAddNew(t *testing.T) {
	dst := Default()
	src := Config{Rules: map[string]RuleConfig{
		"custom_rule_1": {Threshold: 3},
	}}
	merge(dst, &src)
	if _, ok := dst.Rules["custom_rule_1"]; !ok {
		t.Error("new rule should be added")
	}
}

func TestMergeOutputAndParsers(t *testing.T) {
	dst := Default()
	src := Config{
		Output:  OutputConfig{Format: "json"},
		Parsers: map[string]bool{"json": false},
	}
	merge(dst, &src)
	if dst.Output.Format != "json" {
		t.Errorf("Format = %q, want json", dst.Output.Format)
	}
	if dst.Parsers["json"] {
		t.Error("json parser should be disabled")
	}
}

func TestMergeFileFromTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.yaml")
	content := "anomaly:\n  min_errors: 77\noutput:\n  format: markdown\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := mergeFile(cfg, path); err != nil {
		t.Fatal(err)
	}
	if cfg.Anomaly.MinErrors != 77 {
		t.Errorf("MinErrors = %d, want 77", cfg.Anomaly.MinErrors)
	}
	if cfg.Output.Format != "markdown" {
		t.Errorf("Format = %q, want markdown", cfg.Output.Format)
	}
	if cfg.Anomaly.MinBaseline != 5 {
		t.Errorf("MinBaseline = %d, want 5 (unset field preserved)", cfg.Anomaly.MinBaseline)
	}
}

func TestMergeFileMissingIsIgnored(t *testing.T) {
	if err := mergeFile(Default(), filepath.Join(t.TempDir(), "nope.yaml")); err != nil {
		t.Fatalf("missing file must be ignored, got %v", err)
	}
}

func TestMergeFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("anomaly: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := mergeFile(Default(), path); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestUnknownKeysTolerated(t *testing.T) {
	// Unknown keys must not fail loading (forward compatibility).
	dir := t.TempDir()
	path := filepath.Join(dir, "future.yaml")
	content := "some_future_setting:\n  value: 1\nanomaly:\n  min_errors: 9\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := mergeFile(cfg, path); err != nil {
		t.Fatalf("unknown keys must be tolerated: %v", err)
	}
	if cfg.Anomaly.MinErrors != 9 {
		t.Errorf("MinErrors = %d, want 9", cfg.Anomaly.MinErrors)
	}
}

func TestValidateDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestValidateAnomaly(t *testing.T) {
	tests := []struct {
		name string
		cfg  AnomalyConfig
		want string
	}{
		{"zero baseline", AnomalyConfig{MinBaseline: 0, ZScore: 3, MinIncrease: 3, MinErrors: 10}, "min_baseline"},
		{"zero zscore", AnomalyConfig{MinBaseline: 5, ZScore: 0, MinIncrease: 3, MinErrors: 10}, "z_score"},
		{"negative errors", AnomalyConfig{MinBaseline: 5, ZScore: 3, MinIncrease: 3, MinErrors: -1}, "min_errors"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.Anomaly = tt.cfg
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q should mention %q", err.Error(), tt.want)
			}
		})
	}
}

func TestValidateCustomRules(t *testing.T) {
	tests := []struct {
		name string
		rule CustomRuleConfig
		want string
	}{
		{"empty id", CustomRuleConfig{RootCause: "x", Severity: "high", Keywords: []string{"a"}}, "id"},
		{"empty root cause", CustomRuleConfig{ID: "r1", Severity: "high", Keywords: []string{"a"}}, "root_cause"},
		{"no keywords", CustomRuleConfig{ID: "r1", RootCause: "x", Severity: "high"}, "keywords"},
		{"bad severity", CustomRuleConfig{ID: "r1", RootCause: "x", Severity: "urgent", Keywords: []string{"a"}}, "severity"},
		{"weight too high", CustomRuleConfig{ID: "r1", RootCause: "x", Severity: "high", Keywords: []string{"a"}, StrongWeight: 1.5}, "strong_weight"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.CustomRules = []CustomRuleConfig{tt.rule}
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q should mention %q", err.Error(), tt.want)
			}
		})
	}
}

func TestValidateDuplicateRuleID(t *testing.T) {
	cfg := Default()
	cfg.CustomRules = []CustomRuleConfig{
		{ID: "dup", RootCause: "a", Severity: "high", Keywords: []string{"x"}},
		{ID: "dup", RootCause: "b", Severity: "high", Keywords: []string{"y"}},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestValidateOutputFormat(t *testing.T) {
	cfg := Default()
	cfg.Output.Format = "html"
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "output.format") {
		t.Fatalf("expected output.format error, got %v", err)
	}
}
