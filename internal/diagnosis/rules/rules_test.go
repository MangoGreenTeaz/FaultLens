package rules

import (
	"testing"
	"time"

	"github.com/faultlens/faultlens/internal/anomaly"
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/model"
)

func base() time.Time {
	return time.Date(2026, 8, 31, 14, 32, 0, 0, time.UTC)
}

func grp(msg string, count int, first time.Time) grouping.ErrorGroup {
	return grouping.ErrorGroup{
		Message:   msg,
		Count:     count,
		FirstSeen: first,
		LastSeen:  first.Add(time.Minute),
	}
}

func fiveXX(at time.Time) *model.LogEvent {
	return &model.LogEvent{
		Level:     model.LevelError,
		Message:   "GET /api -> 500",
		Fields:    map[string]string{"status": "500"},
		Timestamp: at,
	}
}

func TestDatabaseRule(t *testing.T) {
	dbFirst := base()
	fiveFirst := dbFirst.Add(5 * time.Second)
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("Connection refused <IP>:<PORT>", 50, dbFirst),
			grp("MySQL connection failed", 20, dbFirst),
			grp("HTTP 500", 30, fiveFirst),
		},
		Events: []*model.LogEvent{fiveXX(fiveFirst)},
	}
	d := NewDatabaseRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a database diagnosis")
	}
	if d.Confidence < diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want above threshold", d.Confidence)
	}
	if d.RootCause != "Database unavailable" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	if len(d.Evidence) < 3 {
		t.Errorf("got %d evidence entries, want >= 3 (pattern + downstream + temporal)", len(d.Evidence))
	}
	if len(d.Recommendations) == 0 {
		t.Error("recommendations missing")
	}
}

func TestDatabaseRuleRequiresStrongSignal(t *testing.T) {
	// Generic connection failures without database keywords are not enough
	// to claim a database incident.
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("Connection refused <IP>:<PORT>", 500, base()),
		},
	}
	d := NewDatabaseRule().Evaluate(ctx)
	if d != nil {
		t.Errorf("expected nil for generic connection failures, got %+v", d)
	}
}

func TestRedisRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("redis timeout after <NUMBER>ms", 40, base()),
			grp("Connection refused <IP>:<PORT>", 10, base()),
		},
	}
	d := NewRedisRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a redis diagnosis")
	}
	if d.Confidence < diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want above threshold", d.Confidence)
	}
	if d.RootCause != "Redis unavailable" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
}

func TestRedisRuleNoSignal(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("Connection refused <IP>:<PORT>", 100, base())},
	}
	if d := NewRedisRule().Evaluate(ctx); d != nil {
		t.Errorf("expected nil, got %+v", d)
	}
}

func TestOOMRuleCritical(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("java.lang.OutOfMemoryError: Java heap space", 5, base())},
	}
	d := NewOOMRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected an OOM diagnosis")
	}
	if d.RootCause != "Out of memory" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	if d.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q, want critical", d.Severity)
	}
	if d.Confidence < diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want above threshold", d.Confidence)
	}
}

func TestTimeoutRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("connection timeout connecting to <IP>:<PORT>", 25, base())},
	}
	d := NewTimeoutRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a timeout diagnosis")
	}
	if d.RootCause != "Connection timeout" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	if d.Severity != model.SeverityMedium {
		t.Errorf("Severity = %q, want medium", d.Severity)
	}
}

func TestHTTPRuleStaysBelowThreshold(t *testing.T) {
	// Pure 5xx with anomaly confirmation must stay below the threshold so
	// the engine reports "Insufficient evidence".
	ctx := &diagnosis.DiagnosisContext{
		Events:    []*model.LogEvent{fiveXX(base()), fiveXX(base().Add(time.Second))},
		Anomalies: []anomaly.Detection{{Bucket: base(), Current: 50}},
	}
	d := NewHTTPRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected an http diagnosis")
	}
	if d.Confidence >= diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want below threshold (5xx is a symptom)", d.Confidence)
	}
}

func TestHTTPRuleDowngradesWithUpstream(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("MySQL connection failed", 50, base()),
		},
		Events: []*model.LogEvent{fiveXX(base())},
	}
	d := NewHTTPRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected an http diagnosis")
	}
	if d.Confidence >= diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want below threshold when upstream is down", d.Confidence)
	}
}

func TestCrashRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("panic: runtime error: invalid memory address", 8, base())},
	}
	d := NewCrashRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a crash diagnosis")
	}
	if d.RootCause != "Application crash" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	if d.Severity != model.SeverityHigh {
		t.Errorf("Severity = %q, want high", d.Severity)
	}
	if d.Confidence < diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want above threshold", d.Confidence)
	}
}

func TestCrashRuleDowngradesWithOOM(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("java.lang.OutOfMemoryError: Java heap space", 5, base()),
			grp("application crashed", 1, base()),
		},
	}
	d := NewCrashRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected a crash diagnosis")
	}
	if d.Confidence >= diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v, want below threshold when OOM present", d.Confidence)
	}
}

func TestRegistryRegistersAllRules(t *testing.T) {
	e := diagnosis.NewEngine()
	RegisterDefaultRules(e)
	if e.RuleCount() != 14 {
		t.Errorf("RuleCount = %d, want 14", e.RuleCount())
	}
}
