package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MangoGreenTeaz/FaultLens/internal/engine"
	"github.com/MangoGreenTeaz/FaultLens/internal/grouping"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

func diffResult(events int, groups []grouping.ErrorGroup, root string, conf float64) *engine.Result {
	return &engine.Result{
		Summary:     engine.Summary{Events: events},
		ErrorGroups: groups,
		Diagnosis:   &model.Diagnosis{RootCause: root, Confidence: conf},
	}
}

func dgrp(fp, msg string, count int) grouping.ErrorGroup {
	return grouping.ErrorGroup{Fingerprint: fp, Message: msg, Count: count}
}

func TestComputeDiffSame(t *testing.T) {
	before := diffResult(100, []grouping.ErrorGroup{dgrp("a", "error a", 10)}, "Database unavailable", 0.9)
	after := diffResult(100, []grouping.ErrorGroup{dgrp("a", "error a", 10)}, "Database unavailable", 0.9)

	d := ComputeDiff(before, after)
	if len(d.AddedGroups) != 0 || len(d.RemovedGroups) != 0 || len(d.ChangedGroups) != 0 {
		t.Errorf("identical reports produced differences: %+v", d)
	}
	if d.DiagnosisChanged {
		t.Error("identical diagnosis flagged as changed")
	}
}

func TestComputeDiffAddedRemovedChanged(t *testing.T) {
	before := diffResult(100, []grouping.ErrorGroup{
		dgrp("gone", "error gone", 30),
		dgrp("same", "error same", 5),
	}, "Redis unavailable", 0.71)
	after := diffResult(120, []grouping.ErrorGroup{
		dgrp("new", "error new", 12),
		dgrp("same", "error same", 25),
	}, "Database unavailable", 0.93)

	d := ComputeDiff(before, after)
	if len(d.AddedGroups) != 1 || d.AddedGroups[0].Message != "error new" {
		t.Errorf("added = %+v", d.AddedGroups)
	}
	if len(d.RemovedGroups) != 1 || d.RemovedGroups[0].Message != "error gone" {
		t.Errorf("removed = %+v", d.RemovedGroups)
	}
	if len(d.ChangedGroups) != 1 || d.ChangedGroups[0].Message != "error same" {
		t.Errorf("changed = %+v", d.ChangedGroups)
	}
	if d.ChangedGroups[0].BeforeCount != 5 || d.ChangedGroups[0].AfterCount != 25 {
		t.Errorf("changed counts = %+v", d.ChangedGroups[0])
	}
	if !d.DiagnosisChanged || d.RootCauseBefore != "Redis unavailable" || d.RootCauseAfter != "Database unavailable" {
		t.Errorf("diagnosis change = %+v", d)
	}
	if d.ConfidenceBefore != 0.71 || d.ConfidenceAfter != 0.93 {
		t.Errorf("confidence = %v -> %v", d.ConfidenceBefore, d.ConfidenceAfter)
	}
	if d.EventsBefore != 100 || d.EventsAfter != 120 {
		t.Errorf("events = %d -> %d", d.EventsBefore, d.EventsAfter)
	}
}

func TestComputeDiffConfidenceOnly(t *testing.T) {
	before := diffResult(50, []grouping.ErrorGroup{dgrp("a", "error a", 5)}, "Database unavailable", 0.60)
	after := diffResult(50, []grouping.ErrorGroup{dgrp("a", "error a", 5)}, "Database unavailable", 0.90)

	d := ComputeDiff(before, after)
	if d.DiagnosisChanged {
		t.Error("same root cause must not be flagged as diagnosis change")
	}
	if d.ConfidenceBefore == d.ConfidenceAfter {
		t.Error("confidence change not captured")
	}
}

func TestRenderDiffNoChanges(t *testing.T) {
	d := ComputeDiff(diffResult(10, nil, "Insufficient evidence", 0), diffResult(10, nil, "Insufficient evidence", 0))
	var buf bytes.Buffer
	if err := RenderDiff(&buf, d); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No differences detected") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestRenderDiffChanges(t *testing.T) {
	d := &Diff{
		AddedGroups:      []GroupDiff{{Message: "database unavailable", AfterCount: 12, Status: "added"}},
		RemovedGroups:    []GroupDiff{{Message: "redis timeout", BeforeCount: 30, Status: "removed"}},
		ChangedGroups:    []GroupDiff{{Message: "connection refused", BeforeCount: 5, AfterCount: 25, Status: "changed"}},
		DiagnosisChanged: true,
		RootCauseBefore:  "Redis unavailable",
		RootCauseAfter:   "Database unavailable",
		ConfidenceBefore: 0.71,
		ConfidenceAfter:  0.93,
		EventsBefore:     100,
		EventsAfter:      120,
	}
	var buf bytes.Buffer
	if err := RenderDiff(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{
		"Error Groups", "+ database unavailable (new, count: 12)",
		"- redis timeout (gone, was: 30)",
		"~ connection refused (5 -> 25)",
		"Diagnosis changed:", "Redis unavailable", "Database unavailable",
		"Confidence:", "Events: 100 → 120",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("diff output missing %q; got:\n%s", want, s)
		}
	}
}

func TestRenderDiffMarkdown(t *testing.T) {
	d := &Diff{
		AddedGroups:      []GroupDiff{{Message: "error new", AfterCount: 3, Status: "added"}},
		DiagnosisChanged: true,
		RootCauseBefore:  "Redis unavailable",
		RootCauseAfter:   "Database unavailable",
	}
	var buf bytes.Buffer
	if err := RenderDiffMarkdown(&buf, d); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"## Error Groups", "**+ added**", "**Diagnosis changed:**", "`Redis unavailable` → `Database unavailable`"} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown diff missing %q", want)
		}
	}
}

func TestLoadResultRoundTrip(t *testing.T) {
	before := diffResult(10, []grouping.ErrorGroup{dgrp("a", "error a", 1)}, "Database unavailable", 0.5)
	var buf bytes.Buffer
	if err := RenderJSON(&buf, before); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadResult(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Summary.Events != 10 || len(loaded.ErrorGroups) != 1 || loaded.Diagnosis.RootCause != "Database unavailable" {
		t.Errorf("round-trip mismatch: %+v", loaded)
	}
}

func TestLoadResultInvalid(t *testing.T) {
	if _, err := LoadResult(strings.NewReader("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
