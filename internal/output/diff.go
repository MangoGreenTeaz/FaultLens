package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/MangoGreenTeaz/FaultLens/internal/engine"
)

// GroupDiff describes one error group that appeared, disappeared or changed
// between two reports.
type GroupDiff struct {
	Message     string
	BeforeCount int
	AfterCount  int
	Status      string // "added", "removed" or "changed"
}

// Diff is the comparison between two FaultLens JSON reports.
type Diff struct {
	AddedGroups   []GroupDiff
	RemovedGroups []GroupDiff
	ChangedGroups []GroupDiff

	DiagnosisChanged bool
	RootCauseBefore  string
	RootCauseAfter   string
	ConfidenceBefore float64
	ConfidenceAfter  float64

	EventsBefore int
	EventsAfter  int
}

// LoadResult decodes one FaultLens JSON report from r. It reuses the exact
// JSON schema emitted by RenderJSON, so no separate persistent format exists.
func LoadResult(r io.Reader) (*engine.Result, error) {
	var res engine.Result
	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ComputeDiff compares two reports and classifies every error group as
// added, removed or changed (count differs), plus any diagnosis change.
func ComputeDiff(before, after *engine.Result) *Diff {
	d := &Diff{
		EventsBefore: before.Summary.Events,
		EventsAfter:  after.Summary.Events,
	}

	if before.Diagnosis != nil && after.Diagnosis != nil {
		if before.Diagnosis.RootCause != after.Diagnosis.RootCause {
			d.DiagnosisChanged = true
			d.RootCauseBefore = before.Diagnosis.RootCause
			d.RootCauseAfter = after.Diagnosis.RootCause
		}
		d.ConfidenceBefore = before.Diagnosis.Confidence
		d.ConfidenceAfter = after.Diagnosis.Confidence
	}

	byFingerprint := make(map[string]*GroupDiff, len(before.ErrorGroups)+len(after.ErrorGroups))
	for i := range before.ErrorGroups {
		g := &before.ErrorGroups[i]
		byFingerprint[g.Fingerprint] = &GroupDiff{Message: g.Message, BeforeCount: g.Count}
	}
	for i := range after.ErrorGroups {
		g := &after.ErrorGroups[i]
		if gd, ok := byFingerprint[g.Fingerprint]; ok {
			gd.AfterCount = g.Count
		} else {
			byFingerprint[g.Fingerprint] = &GroupDiff{Message: g.Message, AfterCount: g.Count}
		}
	}

	for _, gd := range byFingerprint {
		switch {
		case gd.BeforeCount == 0:
			gd.Status = "added"
			d.AddedGroups = append(d.AddedGroups, *gd)
		case gd.AfterCount == 0:
			gd.Status = "removed"
			d.RemovedGroups = append(d.RemovedGroups, *gd)
		case gd.BeforeCount != gd.AfterCount:
			gd.Status = "changed"
			d.ChangedGroups = append(d.ChangedGroups, *gd)
		}
	}
	sortByMessage(d.AddedGroups)
	sortByMessage(d.RemovedGroups)
	sortByMessage(d.ChangedGroups)
	return d
}

func sortByMessage(groups []GroupDiff) {
	sort.Slice(groups, func(i, j int) bool { return groups[i].Message < groups[j].Message })
}

// RenderDiff writes the human-readable comparison.
func RenderDiff(w io.Writer, d *Diff) error {
	var b strings.Builder

	if len(d.AddedGroups) == 0 && len(d.RemovedGroups) == 0 &&
		len(d.ChangedGroups) == 0 && !d.DiagnosisChanged {
		b.WriteString("No differences detected\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	if len(d.AddedGroups)+len(d.RemovedGroups)+len(d.ChangedGroups) > 0 {
		b.WriteString("Error Groups\n")
		for _, g := range d.AddedGroups {
			fmt.Fprintf(&b, "+ %s (new, count: %d)\n", g.Message, g.AfterCount)
		}
		for _, g := range d.RemovedGroups {
			fmt.Fprintf(&b, "- %s (gone, was: %d)\n", g.Message, g.BeforeCount)
		}
		for _, g := range d.ChangedGroups {
			fmt.Fprintf(&b, "~ %s (%d -> %d)\n", g.Message, g.BeforeCount, g.AfterCount)
		}
		b.WriteString("\n")
	}

	if d.DiagnosisChanged {
		b.WriteString("Diagnosis changed:\n")
		fmt.Fprintf(&b, "%s\n→\n%s\n\n", d.RootCauseBefore, d.RootCauseAfter)
	}
	if d.ConfidenceBefore != d.ConfidenceAfter {
		fmt.Fprintf(&b, "Confidence:\n%.2f\n→\n%.2f\n\n", d.ConfidenceBefore, d.ConfidenceAfter)
	}
	if d.EventsBefore != d.EventsAfter {
		fmt.Fprintf(&b, "Events: %d → %d\n", d.EventsBefore, d.EventsAfter)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// RenderDiffMarkdown writes the comparison as Markdown (issue / PR comment
// friendly).
func RenderDiffMarkdown(w io.Writer, d *Diff) error {
	var b strings.Builder

	if len(d.AddedGroups) == 0 && len(d.RemovedGroups) == 0 &&
		len(d.ChangedGroups) == 0 && !d.DiagnosisChanged {
		b.WriteString("No differences detected\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	b.WriteString("## Error Groups\n\n")
	b.WriteString("| Change | Error | Count |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, g := range d.AddedGroups {
		fmt.Fprintf(&b, "| **+ added** | %s | %d |\n", g.Message, g.AfterCount)
	}
	for _, g := range d.RemovedGroups {
		fmt.Fprintf(&b, "| **- removed** | %s | was %d |\n", g.Message, g.BeforeCount)
	}
	for _, g := range d.ChangedGroups {
		fmt.Fprintf(&b, "| **~ changed** | %s | %d → %d |\n", g.Message, g.BeforeCount, g.AfterCount)
	}

	if d.DiagnosisChanged {
		b.WriteString("\n**Diagnosis changed:**\n\n")
		fmt.Fprintf(&b, "`%s` → `%s`\n", d.RootCauseBefore, d.RootCauseAfter)
	}
	if d.ConfidenceBefore != d.ConfidenceAfter {
		fmt.Fprintf(&b, "\n**Confidence:** %.2f → %.2f\n", d.ConfidenceBefore, d.ConfidenceAfter)
	}
	if d.EventsBefore != d.EventsAfter {
		fmt.Fprintf(&b, "\n**Events:** %d → %d\n", d.EventsBefore, d.EventsAfter)
	}

	_, err := io.WriteString(w, b.String())
	return err
}
