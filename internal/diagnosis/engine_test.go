package diagnosis_test

import (
	"testing"
	"time"

	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/diagnosis/rules"
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

func engineWithDefaultRules() *diagnosis.Engine {
	e := diagnosis.NewEngine()
	rules.RegisterDefaultRules(e)
	return e
}

func TestEngineEmptyReturnsInsufficient(t *testing.T) {
	d := diagnosis.NewEngine().Diagnose(&diagnosis.DiagnosisContext{})
	if d.RootCause != "Insufficient evidence" {
		t.Errorf("RootCause = %q, want Insufficient evidence", d.RootCause)
	}
	if d.Confidence != 0 {
		t.Errorf("Confidence = %v, want 0", d.Confidence)
	}
	if d.Severity != model.SeverityLow {
		t.Errorf("Severity = %q, want low", d.Severity)
	}
}

func TestEngineNoEvidenceReturnsInsufficient(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("user logged in", 100, base()),
		},
	}
	d := engineWithDefaultRules().Diagnose(ctx)
	if d.RootCause != "Insufficient evidence" {
		t.Errorf("RootCause = %q, want Insufficient evidence", d.RootCause)
	}
}

func TestEnginePrefersDatabaseOverHTTP(t *testing.T) {
	// MySQL down + HTTP 5xx downstream: the engine must pick the database,
	// never the symptom.
	dbFirst := base()
	fiveFirst := base().Add(5 * time.Second)
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("Connection refused <IP>:<PORT>", 50, dbFirst),
			grp("MySQL connection failed", 20, dbFirst),
			grp("HTTP 500", 30, fiveFirst),
		},
		Events: []*model.LogEvent{fiveXX(fiveFirst)},
	}
	d := engineWithDefaultRules().Diagnose(ctx)
	if d.RootCause != "Database unavailable" {
		t.Fatalf("RootCause = %q, want Database unavailable", d.RootCause)
	}
	if d.Confidence < 0.5 {
		t.Errorf("Confidence = %v, want >= 0.5 for strong db evidence", d.Confidence)
	}
	if d.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q, want critical", d.Severity)
	}
	if len(d.Evidence) == 0 {
		t.Error("diagnosis must carry evidence")
	}
	if len(d.Recommendations) == 0 {
		t.Error("diagnosis must carry recommendations")
	}
}

func TestEnginePureHTTP5xxIsInsufficient(t *testing.T) {
	// No upstream cause: the engine must NOT guess MySQL/Redis/database.
	ctx := &diagnosis.DiagnosisContext{
		Events: []*model.LogEvent{
			fiveXX(base()),
			fiveXX(base().Add(time.Second)),
			fiveXX(base().Add(2 * time.Second)),
		},
	}
	d := engineWithDefaultRules().Diagnose(ctx)
	if d.RootCause != "Insufficient evidence" {
		t.Fatalf("RootCause = %q, want Insufficient evidence (5xx is a symptom)", d.RootCause)
	}
	// The candidate evidence must still be available for explainability.
	if len(d.Evidence) == 0 {
		t.Error("insufficient diagnosis should keep candidate evidence")
	}
}

func TestEnginePrefersOOMOverCrash(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("java.lang.OutOfMemoryError: Java heap space", 3, base()),
			grp("application crashed", 1, base()),
		},
	}
	d := engineWithDefaultRules().Diagnose(ctx)
	if d.RootCause != "Out of memory" {
		t.Fatalf("RootCause = %q, want Out of memory", d.RootCause)
	}
	if d.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q, want critical", d.Severity)
	}
}

func TestEngineConfidenceClamped(t *testing.T) {
	if got := diagnosis.ClampConfidence(-1); got != 0 {
		t.Errorf("ClampConfidence(-1) = %v, want 0", got)
	}
	if got := diagnosis.ClampConfidence(2); got != 1 {
		t.Errorf("ClampConfidence(2) = %v, want 1", got)
	}
	if got := diagnosis.ClampConfidence(0.5); got != 0.5 {
		t.Errorf("ClampConfidence(0.5) = %v, want 0.5", got)
	}
}
