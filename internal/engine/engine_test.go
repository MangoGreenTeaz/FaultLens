package engine

import (
	"strings"
	"testing"
	"time"
)

func TestRunBasicAnalysis(t *testing.T) {
	logs := `2026-08-31 14:32:01 ERROR database connection failed
2026-08-31 14:32:02 ERROR database connection failed
2026-08-31 14:32:03 INFO request handled
`
	res, err := Run(strings.NewReader(logs), Options{Source: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Events != 3 {
		t.Errorf("Events = %d, want 3", res.Summary.Events)
	}
	if res.Summary.Errors != 2 {
		t.Errorf("Errors = %d, want 2", res.Summary.Errors)
	}
	if res.Summary.Warnings != 0 {
		t.Errorf("Warnings = %d, want 0", res.Summary.Warnings)
	}
	if res.Summary.Format != "java" {
		t.Errorf("Format = %q, want java (timestamped lines with levels)", res.Summary.Format)
	}
	if res.Summary.Source != "test" {
		t.Errorf("Source = %q, want test", res.Summary.Source)
	}
	if len(res.ErrorGroups) != 1 {
		t.Errorf("ErrorGroups = %d, want 1 (identical errors merge)", len(res.ErrorGroups))
	}
	if res.Diagnosis == nil {
		t.Error("Diagnosis should never be nil")
	}
}

func TestRunAutoDetectJSON(t *testing.T) {
	logs := `{"timestamp":"2026-08-31T14:32:01Z","level":"ERROR","message":"db down","service":"api"}
{"timestamp":"2026-08-31T14:32:02Z","level":"INFO","message":"ok","service":"api"}
`
	res, err := Run(strings.NewReader(logs), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Format != "json" {
		t.Errorf("Format = %q, want json", res.Summary.Format)
	}
	if res.Summary.Errors != 1 {
		t.Errorf("Errors = %d, want 1", res.Summary.Errors)
	}
}

func TestRunFormatOverride(t *testing.T) {
	// With --format plain, arbitrary text becomes UNKNOWN events, not parse
	// failures.
	res, err := Run(strings.NewReader("just a line without timestamp\n"), Options{Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Events != 1 {
		t.Errorf("Events = %d, want 1", res.Summary.Events)
	}
	if res.Summary.ParsingWarnings != 0 {
		t.Errorf("ParsingWarnings = %d, want 0", res.Summary.ParsingWarnings)
	}
}

func TestRunTimeFilter(t *testing.T) {
	logs := `2026-08-31 14:00:00 ERROR before
2026-08-31 14:30:00 ERROR inside
2026-08-31 15:00:00 ERROR after
`
	from := time.Date(2026, 8, 31, 14, 1, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 14, 59, 0, 0, time.UTC)
	res, err := Run(strings.NewReader(logs), Options{From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Events != 1 {
		t.Errorf("Events = %d, want 1 (only the middle event)", res.Summary.Events)
	}
	if !res.Summary.FirstEvent.Equal(time.Date(2026, 8, 31, 14, 30, 0, 0, time.UTC)) {
		t.Errorf("FirstEvent = %v, want 14:30:00", res.Summary.FirstEvent)
	}
}

func TestRunEmptyInput(t *testing.T) {
	res, err := Run(strings.NewReader(""), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Events != 0 {
		t.Errorf("Events = %d, want 0", res.Summary.Events)
	}
	if res.Diagnosis.RootCause != "Insufficient evidence" {
		t.Errorf("RootCause = %q, want Insufficient evidence", res.Diagnosis.RootCause)
	}
}

func TestRunJSONWarnings(t *testing.T) {
	logs := `{"level":"ERROR","message":"boom"}
this is not json
{"level":"INFO","message":"ok"}
`
	res, err := Run(strings.NewReader(logs), Options{Format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Events != 2 {
		t.Errorf("Events = %d, want 2 (malformed line produces no event)", res.Summary.Events)
	}
	if res.Summary.ParsingWarnings != 1 {
		t.Errorf("ParsingWarnings = %d, want 1", res.Summary.ParsingWarnings)
	}
}

func TestRunMySQLIncidentDiagnosis(t *testing.T) {
	// Simulated outage chain: connection refused → app errors → HTTP 500.
	var b strings.Builder
	for i := 0; i < 10; i++ {
		ts := time.Date(2026, 8, 31, 14, 32, i%10, 0, time.UTC).Format("2006-01-02 15:04:05")
		b.WriteString(ts + " ERROR Connection refused 10.0.0.1:3306\n")
	}
	for i := 0; i < 10; i++ {
		ts := time.Date(2026, 8, 31, 14, 32, 10+i%10, 0, time.UTC).Format("2006-01-02 15:04:05")
		b.WriteString(ts + " ERROR MySQL connection failed\n")
	}
	for i := 0; i < 10; i++ {
		ts := time.Date(2026, 8, 31, 14, 32, 20+i%10, 0, time.UTC).Format("2006-01-02 15:04:05")
		b.WriteString(ts + " ERROR HTTP 500 Internal Server Error\n")
	}

	res, err := Run(strings.NewReader(b.String()), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Diagnosis.RootCause != "Database unavailable" {
		t.Errorf("RootCause = %q, want Database unavailable", res.Diagnosis.RootCause)
	}
	if res.Diagnosis.Confidence < 0.5 {
		t.Errorf("Confidence = %v, want >= 0.5", res.Diagnosis.Confidence)
	}
	if len(res.Diagnosis.Evidence) == 0 {
		t.Error("diagnosis must carry evidence")
	}
}
