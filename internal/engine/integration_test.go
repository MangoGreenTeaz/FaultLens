package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIncidentFixturesDiagnosis runs the full pipeline over every incident
// fixture and asserts the expected root cause. These are the acceptance
// scenarios from the spec: the diagnosis must come from log evidence, not
// from the fixture file name.
func TestIncidentFixturesDiagnosis(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"mysql-outage.log", "Database unavailable"},
		{"redis-outage.log", "Redis unavailable"},
		{"oom.log", "Out of memory"},
		{"http-5xx.log", "Insufficient evidence"},
		{"application-crash.log", "Application crash"},
		{"disk-full.log", "Disk full"},
		{"certificate-expired.log", "Certificate expired"},
		{"mq-outage.log", "Message queue unavailable"},
		{"connection-pool-exhausted.log", "Connection pool exhausted"},
		{"network-partition.log", "Network partition"},
		{"cpu-saturation.log", "CPU saturation"},
		{"slow-query.log", "Slow query"},
		{"deadlock.log", "Deadlock detected"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "incidents", tt.file)
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open %s: %v", path, err)
			}
			defer f.Close()

			res, err := Run(f, Options{Source: tt.file})
			if err != nil {
				t.Fatal(err)
			}
			if res.Summary.Events == 0 {
				t.Error("fixture produced no events")
			}
			if res.Diagnosis.Confidence < 0 || res.Diagnosis.Confidence > 1 {
				t.Errorf("confidence out of range: %v", res.Diagnosis.Confidence)
			}
			if res.Diagnosis.RootCause != tt.want {
				t.Errorf("root cause = %q, want %q (confidence %.2f, %d evidence)",
					res.Diagnosis.RootCause, tt.want, res.Diagnosis.Confidence, len(res.Diagnosis.Evidence))
			}
		})
	}
}

// TestMySQLOutageEvidence verifies the explainability requirement: the MySQL
// diagnosis must carry real evidence and the 5xx symptom must not win.
func TestMySQLOutageEvidence(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "testdata", "incidents", "mysql-outage.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	res, err := Run(f, Options{Source: "mysql-outage.log"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnosis.Evidence) == 0 {
		t.Error("diagnosis must carry evidence")
	}
	if res.Diagnosis.RootCause != "Database unavailable" {
		t.Fatalf("root cause = %q, must never report the 5xx symptom instead", res.Diagnosis.RootCause)
	}
	// 5xx is a symptom: it should appear as downstream evidence, not as the
	// winning hypothesis.
	if res.Diagnosis.Confidence < 0.5 {
		t.Errorf("confidence = %v, want >= 0.5 for the mysql chain", res.Diagnosis.Confidence)
	}
}

// TestFormatFixturesDetected checks content-based format detection.
func TestFormatFixturesDetected(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"plain/basic.log", "plain"},
		{"json/basic.jsonl", "json"},
		{"java/spring.log", "java"},
		{"nginx/access.log", "nginx"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", tt.file)
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open %s: %v", path, err)
			}
			defer f.Close()

			res, err := Run(f, Options{Source: tt.file})
			if err != nil {
				t.Fatal(err)
			}
			if res.Summary.Format != tt.want {
				t.Errorf("detected format = %q, want %q", res.Summary.Format, tt.want)
			}
			if res.Summary.Events == 0 {
				t.Error("fixture produced no events")
			}
		})
	}
}

// TestJavaFixtureAggregatesStackTrace verifies multi-line stack trace
// aggregation through the full pipeline.
func TestJavaFixtureAggregatesStackTrace(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "testdata", "java", "spring.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	res, err := Run(f, Options{Source: "spring.log"})
	if err != nil {
		t.Fatal(err)
	}
	// 3 logical events: the exception + stack trace, an INFO line, a WARN line.
	if res.Summary.Events != 3 {
		t.Errorf("Events = %d, want 3 (stack trace must aggregate into ONE event)", res.Summary.Events)
	}
	for _, g := range res.ErrorGroups {
		for _, ex := range g.Examples {
			if ex.StackTrace != "" {
				return // found an aggregated stack trace
			}
		}
	}
	t.Error("no event carried an aggregated stack trace")
}
