package rules

import (
	"math"
	"testing"
	"time"

	"github.com/faultlens/faultlens/internal/config"
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/model"
)

// base, grp and fiveXX are shared with rules_test.go (same package).

func TestCustomRuleHit(t *testing.T) {
	rule := &CustomRule{
		id:           "disk_full",
		RootCause:    "Disk full",
		Severity:     model.SeverityCritical,
		Keywords:     []string{"no space left on device"},
		StrongWeight: 0.40,
	}
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("no space left on device", 5, base())},
	}
	d := rule.Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a diagnosis")
	}
	if d.RootCause != "Disk full" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	if d.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q, want critical", d.Severity)
	}
	if d.Confidence != 0.40 {
		t.Errorf("Confidence = %v, want 0.40", d.Confidence)
	}
	if len(d.Evidence) == 0 {
		t.Error("evidence missing")
	}
}

func TestCustomRuleNoHit(t *testing.T) {
	rule := &CustomRule{
		id:        "disk_full",
		RootCause: "Disk full",
		Keywords:  []string{"no space left on device"},
	}
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("user logged in", 10, base())},
	}
	if d := rule.Evaluate(ctx); d != nil {
		t.Errorf("expected nil for no match, got %+v", d)
	}
}

func TestCustomRuleSupportingEvidence(t *testing.T) {
	rule := &CustomRule{
		id:           "disk_full",
		RootCause:    "Disk full",
		Keywords:     []string{"no space left on device"},
		StrongWeight: 0.40,
		SupportingKw: []string{"write error"},
		SupportingWt: 0.20,
	}
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("no space left on device", 5, base()),
			grp("write error", 3, base()),
		},
	}
	d := rule.Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a diagnosis")
	}
	if math.Abs(d.Confidence-0.60) > 1e-9 {
		t.Errorf("Confidence = %v, want 0.60", d.Confidence)
	}
	if len(d.Evidence) != 2 {
		t.Errorf("Evidence = %d, want 2", len(d.Evidence))
	}
}

func TestCustomRuleDownstreamAndTemporal(t *testing.T) {
	rule := &CustomRule{
		id:               "mq_unavailable",
		RootCause:        "Message queue unavailable",
		Severity:         model.SeverityCritical,
		Keywords:         []string{"rabbitmq"},
		StrongWeight:     0.40,
		EnableDownstream: true,
	}
	dbFirst := base()
	fiveFirst := base().Add(5 * time.Second)
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("rabbitmq connection failed", 10, dbFirst)},
		FiveXXCount: 1, FiveXXFirst: fiveFirst,
	}
	d := rule.Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a diagnosis")
	}
	// strong 0.40 + downstream 0.15 + temporal 0.15 = 0.70
	if math.Abs(d.Confidence-0.70) > 1e-9 {
		t.Errorf("Confidence = %v, want 0.70", d.Confidence)
	}
	if len(d.Evidence) != 3 {
		t.Errorf("Evidence = %d, want 3", len(d.Evidence))
	}
}

func TestBuildCustomRuleDefaults(t *testing.T) {
	rule, err := buildCustomRule(config.CustomRuleConfig{
		ID:        "r1",
		RootCause: "x",
		Severity:  "high",
		Keywords:  []string{"a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.StrongWeight != defaultStrongWeight {
		t.Errorf("StrongWeight = %v, want %v", rule.StrongWeight, defaultStrongWeight)
	}
	if rule.SupportingWt != defaultSupportingWeight {
		t.Errorf("SupportingWt = %v, want %v", rule.SupportingWt, defaultSupportingWeight)
	}
}

func TestBuildCustomRuleInvalid(t *testing.T) {
	tests := []config.CustomRuleConfig{
		{RootCause: "x", Severity: "high", Keywords: []string{"a"}},
		{ID: "r1", Severity: "high", Keywords: []string{"a"}},
		{ID: "r1", RootCause: "x", Severity: "high"},
		{ID: "r1", RootCause: "x", Severity: "urgent", Keywords: []string{"a"}},
		{ID: "r1", RootCause: "x", Severity: "high", Keywords: []string{"a"}, StrongWeight: 2},
		{ID: "r1", RootCause: "x", Severity: "high", Keywords: []string{"a"}, SupportingWt: -1},
	}
	for i, cr := range tests {
		if _, err := buildCustomRule(cr); err == nil {
			t.Errorf("case %d: expected error for %+v", i, cr)
		}
	}
}

func TestRegisterCustomRulesSkipsInvalid(t *testing.T) {
	e := diagnosis.NewEngine()
	cfgs := []config.CustomRuleConfig{
		{ID: "", RootCause: "broken", Severity: "high", Keywords: []string{"a"}},
		{ID: "disk_full", RootCause: "Disk full", Severity: "critical", Keywords: []string{"no space left"}},
	}
	added, warnings := RegisterCustomRules(e, cfgs)
	if added != 1 {
		t.Errorf("added = %d, want 1", added)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings = %d, want 1", len(warnings))
	}
	if e.RuleCount() != 1 {
		t.Errorf("RuleCount = %d, want 1 (invalid rule skipped)", e.RuleCount())
	}
}
