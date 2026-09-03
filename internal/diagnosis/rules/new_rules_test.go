package rules

import (
	"testing"
	"time"

	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/model"
	"github.com/faultlens/faultlens/internal/timeline"
)

func TestDiskFullRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("no space left on device", 12, base())},
	}
	d := NewDiskFullRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Disk full" || d.Severity != model.SeverityCritical {
		t.Errorf("got %q / %q", d.RootCause, d.Severity)
	}
	if d.Confidence < diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v", d.Confidence)
	}
}

func TestDiskFullRuleNoMatch(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("user logged in", 100, base())},
	}
	if d := NewDiskFullRule().Evaluate(ctx); d != nil {
		t.Errorf("expected nil, got %+v", d)
	}
}

func TestCertificateExpiredRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("SSL certificate expired", 12, base())},
	}
	d := NewCertificateExpiredRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Certificate expired" || d.Severity != model.SeverityHigh {
		t.Errorf("got %q / %q", d.RootCause, d.Severity)
	}
}

func TestMQUnavailableRuleWithDownstream(t *testing.T) {
	mqFirst := base()
	fiveFirst := base().Add(5 * time.Second)
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("rabbitmq connection failed", 12, mqFirst),
			grp("Connection refused <IP>:<PORT>", 6, mqFirst),
		},
		FiveXXCount: 1, FiveXXFirst: fiveFirst,
	}
	d := NewMQUnavailableRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Message queue unavailable" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	// strong 0.40 + volume 0.20 + weak 0.20 + downstream 0.15 + temporal 0.15
	if d.Confidence < 0.9 {
		t.Errorf("Confidence = %v, want >= 0.9", d.Confidence)
	}
	if d.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q", d.Severity)
	}
}

func TestMQUnavailableRuleNoMatch(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("Connection refused <IP>:<PORT>", 100, base())},
	}
	if d := NewMQUnavailableRule().Evaluate(ctx); d != nil {
		t.Errorf("generic connection failures must not trigger MQ, got %+v", d)
	}
}

func TestConnectionPoolExhaustedRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("connection pool exhausted", 12, base())},
	}
	d := NewConnectionPoolExhaustedRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Connection pool exhausted" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	if d.Confidence < diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("Confidence = %v", d.Confidence)
	}
}

func TestConnectionPoolDowngradedByDatabase(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{
			grp("connection pool exhausted", 12, base()),
			grp("MySQL connection failed", 20, base()),
		},
	}
	d := NewConnectionPoolExhaustedRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.Confidence >= diagnosis.InsufficientEvidenceThreshold {
		t.Errorf("pool rule should drop below threshold when db is down, got %v", d.Confidence)
	}
}

func TestNetworkPartitionRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("network unreachable <IP>", 12, base())},
	}
	d := NewNetworkPartitionRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Network partition" || d.Severity != model.SeverityCritical {
		t.Errorf("got %q / %q", d.RootCause, d.Severity)
	}
}

func TestCPUSaturationRuleWithTimeline(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("cpu usage <NUMBER>%", 29, base())},
		Timeline: []timeline.Bucket{
			{Start: base(), Errors: 1},
			{Start: base().Add(time.Minute), Errors: 1},
			{Start: base().Add(2 * time.Minute), Errors: 1},
			{Start: base().Add(3 * time.Minute), Errors: 1},
			{Start: base().Add(4 * time.Minute), Errors: 1},
			{Start: base().Add(5 * time.Minute), Errors: 8},
			{Start: base().Add(6 * time.Minute), Errors: 8},
			{Start: base().Add(7 * time.Minute), Errors: 8},
		},
	}
	d := NewCPUSaturationRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "CPU saturation" {
		t.Errorf("RootCause = %q", d.RootCause)
	}
	// strong 0.40 + temporal (ratio 24/5 >= 2) 0.15
	if d.Confidence < 0.5 {
		t.Errorf("Confidence = %v, want >= 0.5 with timeline signal", d.Confidence)
	}
}

func TestCPUSaturationRuleNoTimelineSignal(t *testing.T) {
	// Flat timeline: temporal bonus must not apply.
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("cpu usage <NUMBER>%", 8, base())},
		Timeline: []timeline.Bucket{
			{Start: base(), Errors: 2},
			{Start: base().Add(time.Minute), Errors: 2},
			{Start: base().Add(2 * time.Minute), Errors: 2},
			{Start: base().Add(3 * time.Minute), Errors: 2},
		},
	}
	d := NewCPUSaturationRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.Confidence >= 0.55 {
		t.Errorf("flat timeline must not add temporal bonus, got %v", d.Confidence)
	}
}

func TestSlowQueryRuleIsSymptom(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("slow query detected", 22, base())},
	}
	d := NewSlowQueryRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Slow query" || d.Severity != model.SeverityMedium {
		t.Errorf("got %q / %q", d.RootCause, d.Severity)
	}
	// Symptom rules start from supporting weight: 0.20 + volume 0.20.
	if d.Confidence > 0.6 {
		t.Errorf("symptom rule confidence too high: %v", d.Confidence)
	}
}

func TestDeadlockRule(t *testing.T) {
	ctx := &diagnosis.DiagnosisContext{
		ErrorGroups: []grouping.ErrorGroup{grp("deadlock detected", 12, base())},
	}
	d := NewDeadlockRule().Evaluate(ctx)
	if d == nil {
		t.Fatal("expected diagnosis")
	}
	if d.RootCause != "Deadlock detected" || d.Severity != model.SeverityHigh {
		t.Errorf("got %q / %q", d.RootCause, d.Severity)
	}
}

func TestTimeSeriesIncrease(t *testing.T) {
	tests := []struct {
		name string
		tl   []timeline.Bucket
		want float64
	}{
		{"empty", nil, 0},
		{"single bucket", []timeline.Bucket{{Errors: 5}}, 0},
		{"rising", []timeline.Bucket{{Errors: 1}, {Errors: 1}, {Errors: 8}, {Errors: 8}}, 8},
		{"flat", []timeline.Bucket{{Errors: 2}, {Errors: 2}, {Errors: 2}, {Errors: 2}}, 1},
		{"zero baseline", []timeline.Bucket{{Errors: 0}, {Errors: 0}, {Errors: 5}, {Errors: 5}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeSeriesIncrease(tt.tl); got != tt.want {
				t.Errorf("timeSeriesIncrease() = %v, want %v", got, tt.want)
			}
		})
	}
}
