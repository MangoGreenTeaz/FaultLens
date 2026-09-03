package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/faultlens/faultlens/internal/anomaly"
	"github.com/faultlens/faultlens/internal/engine"
	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/model"
	"github.com/faultlens/faultlens/internal/timeline"
)

func sampleResult() *engine.Result {
	base := time.Date(2026, 8, 31, 14, 32, 0, 0, time.UTC)
	return &engine.Result{
		Summary: engine.Summary{
			Events:     182391,
			Errors:     4381,
			Warnings:   8212,
			FirstEvent: base,
			LastEvent:  base.Add(time.Hour),
			Format:     "java",
			Source:     "app.log",
		},
		ErrorGroups: []grouping.ErrorGroup{
			{
				Message:   "Connection refused <IP>:<PORT>",
				Count:     4381,
				FirstSeen: base,
				LastSeen:  base.Add(time.Hour),
			},
		},
		Timeline: []timeline.Bucket{
			{Start: base, Total: 100, Errors: 2},
			{Start: base.Add(time.Minute), Total: 1800, Errors: 942},
		},
		Anomalies: []anomaly.Detection{
			{Bucket: base.Add(time.Minute), BaselineMean: 12.4, Current: 942, Increase: 75.9},
		},
		Diagnosis: &model.Diagnosis{
			RootCause:  "Database unavailable",
			Confidence: 0.91,
			Severity:   model.SeverityCritical,
			Evidence: []model.Evidence{
				{Timestamp: base, Type: model.EvidenceErrorPattern, Message: "MySQL connection refused"},
			},
			Recommendations: []string{"Check MySQL availability"},
		},
	}
}

func TestRenderTerminalReport(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderTerminal(&buf, sampleResult(), "report"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{
		"FaultLens", "Log Summary", "182,391", "4,381",
		"Anomalies", "Top Error Groups", "Connection refused <IP>:<PORT>",
		"Diagnosis", "Database unavailable", "0.91", "Recommendations",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("terminal report missing %q", want)
		}
	}
}

func TestRenderTerminalIncident(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderTerminal(&buf, sampleResult(), "incident"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"Incident detected", "Root Cause:", "Database unavailable", "Confidence:", "91%", "Evidence:", "Recommended:"} {
		if !strings.Contains(s, want) {
			t.Errorf("terminal incident missing %q", want)
		}
	}
}

func TestRenderTerminalInsufficient(t *testing.T) {
	res := sampleResult()
	res.Diagnosis = &model.Diagnosis{
		RootCause:  "Insufficient evidence",
		Confidence: 0.21,
		Severity:   model.SeverityLow,
		Evidence:   []model.Evidence{{Type: model.EvidenceErrorPattern, Message: "2 database-related errors"}},
	}
	var buf bytes.Buffer
	if err := RenderTerminal(&buf, res, "incident"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Insufficient evidence") {
		t.Error("insufficient incident view must say so")
	}
	if !strings.Contains(buf.String(), "2 database-related errors") {
		t.Error("insufficient incident view must keep candidate evidence")
	}
}

func TestRenderTerminalErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderTerminal(&buf, sampleResult(), "errors"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"ERROR GROUPS", "Connection refused <IP>:<PORT>", "Occurrences:", "4,381"} {
		if !strings.Contains(s, want) {
			t.Errorf("terminal errors missing %q", want)
		}
	}
}

func TestRenderTerminalTimeline(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderTerminal(&buf, sampleResult(), "timeline"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"TIMELINE", "total=100", "ANOMALY DETECTED", "Baseline error count:", "75.9x"} {
		if !strings.Contains(s, want) {
			t.Errorf("terminal timeline missing %q", want)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, sampleResult()); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, k := range []string{"summary", "error_groups", "timeline", "anomalies", "diagnosis"} {
		if _, ok := m[k]; !ok {
			t.Errorf("JSON missing top-level key %q", k)
		}
	}
	diag, _ := m["diagnosis"].(map[string]any)
	if diag["root_cause"] != "Database unavailable" {
		t.Errorf("diagnosis.root_cause = %v", diag["root_cause"])
	}
	if diag["severity"] != "critical" {
		t.Errorf("diagnosis.severity = %v", diag["severity"])
	}
}

func TestRenderMarkdownReport(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderMarkdown(&buf, sampleResult(), "report"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"# FaultLens Report", "## Log Summary", "## Top Error Groups", "## Diagnosis", "Database unavailable", "Check MySQL availability"} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown report missing %q", want)
		}
	}
}

func TestRenderMarkdownIncident(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderMarkdown(&buf, sampleResult(), "incident"); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"# Incident Report", "**Root Cause:** Database unavailable", "## Evidence", "## Recommended Actions"} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown incident missing %q", want)
		}
	}
}

func TestRenderHTML(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, sampleResult()); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{
		"<!DOCTYPE html>", "FaultLens Report", "Summary", "Diagnosis", "Error Groups",
		"Database unavailable", "182391", "Connection refused &lt;IP&gt;:&lt;PORT&gt;",
		"Evidence", "Check MySQL availability",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("HTML report missing %q", want)
		}
	}
}

func TestRenderHTMLOfflineNoExternalResources(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, sampleResult()); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	// The report must be fully self-contained for CI artifacts and offline
	// debugging: no external URLs, no CDN references.
	for _, forbid := range []string{"http://", "https://", "cdn.", "<script"} {
		if strings.Contains(s, forbid) {
			t.Errorf("HTML report must not reference external resources, found %q", forbid)
		}
	}
	if !strings.Contains(s, "<style>") {
		t.Error("HTML report must inline its CSS")
	}
}

func TestRenderHTMLInsufficient(t *testing.T) {
	res := sampleResult()
	res.Diagnosis = &model.Diagnosis{
		RootCause:  "Insufficient evidence",
		Confidence: 0.21,
		Severity:   model.SeverityLow,
	}
	var buf bytes.Buffer
	if err := RenderHTML(&buf, res); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Insufficient evidence") {
		t.Error("HTML must surface insufficient evidence")
	}
}

func TestComma(t *testing.T) {
	if got := comma(0); got != "0" {
		t.Errorf("comma(0) = %q", got)
	}
	if got := comma(1234); got != "1,234" {
		t.Errorf("comma(1234) = %q, want 1,234", got)
	}
	if got := comma(1234567); got != "1,234,567" {
		t.Errorf("comma(1234567) = %q, want 1,234,567", got)
	}
	if got := comma(182391); got != "182,391" {
		t.Errorf("comma(182391) = %q, want 182,391", got)
	}
}
