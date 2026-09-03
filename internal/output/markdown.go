package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/engine"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// RenderMarkdown writes the report as Markdown suitable for GitHub issues,
// incident reports, postmortems and PR comments.
func RenderMarkdown(w io.Writer, res *engine.Result, kind string) error {
	switch kind {
	case "errors":
		return markdownErrors(w, res)
	case "timeline":
		return markdownTimeline(w, res)
	case "incident":
		return markdownIncident(w, res)
	default:
		return markdownReport(w, res)
	}
}

func markdownReport(w io.Writer, res *engine.Result) error {
	var b strings.Builder
	b.WriteString("# FaultLens Report\n\n")

	b.WriteString("## Log Summary\n\n")
	fmt.Fprintf(&b, "- **Events:** %s\n", comma(res.Summary.Events))
	fmt.Fprintf(&b, "- **Errors:** %s\n", comma(res.Summary.Errors))
	fmt.Fprintf(&b, "- **Warnings:** %s\n", comma(res.Summary.Warnings))
	if !res.Summary.FirstEvent.IsZero() {
		fmt.Fprintf(&b, "- **Time Range:** %s - %s\n",
			res.Summary.FirstEvent.Format("15:04:05"), res.Summary.LastEvent.Format("15:04:05"))
	}
	fmt.Fprintf(&b, "- **Format:** %s\n", res.Summary.Format)
	if res.Summary.ParsingWarnings > 0 {
		fmt.Fprintf(&b, "- **Parsing warnings:** %d lines could not be fully parsed\n", res.Summary.ParsingWarnings)
	}

	b.WriteString("\n## Anomalies\n\n")
	if len(res.Anomalies) == 0 {
		b.WriteString("None detected\n")
	} else {
		b.WriteString("| Time | Baseline | Current | Increase |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, a := range res.Anomalies {
			fmt.Fprintf(&b, "| %s | %.1f | %d | %.1fx |\n",
				a.Bucket.Format("15:04"), a.BaselineMean, a.Current, a.Increase)
		}
	}

	b.WriteString("\n## Top Error Groups\n\n")
	if len(res.ErrorGroups) == 0 {
		b.WriteString("No error groups\n")
	} else {
		b.WriteString("| # | Error | Occurrences |\n")
		b.WriteString("| --- | --- | --- |\n")
		for i := 0; i < min(5, len(res.ErrorGroups)); i++ {
			g := res.ErrorGroups[i]
			fmt.Fprintf(&b, "| %d | %s | %s |\n", i+1, g.Message, comma(g.Count))
		}
	}

	b.WriteString("\n## Diagnosis\n\n")
	markdownDiagnosis(&b, res.Diagnosis)

	_, err := io.WriteString(w, b.String())
	return err
}

func markdownErrors(w io.Writer, res *engine.Result) error {
	var b strings.Builder
	b.WriteString("# Error Groups\n\n")
	if len(res.ErrorGroups) == 0 {
		b.WriteString("No error groups\n")
	} else {
		b.WriteString("| # | Error | Occurrences | First seen | Last seen |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for i, g := range res.ErrorGroups {
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n", i+1, g.Message, comma(g.Count),
				formatClock(g.FirstSeen), formatClock(g.LastSeen))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func markdownTimeline(w io.Writer, res *engine.Result) error {
	var b strings.Builder
	b.WriteString("# Timeline\n\n")
	if len(res.Timeline) == 0 {
		b.WriteString("No timeline data\n")
	} else {
		b.WriteString("| Time | Total | Errors | Warnings | Fatal |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, tb := range res.Timeline {
			fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n",
				tb.Start.Format("15:04"), tb.Total, tb.Errors, tb.Warnings, tb.Fatal)
		}
	}
	if len(res.Anomalies) > 0 {
		b.WriteString("\n## Anomalies\n\n")
		for _, a := range res.Anomalies {
			fmt.Fprintf(&b, "- **%s**: error count %d vs baseline %.1f (%.1fx, z=%.2f)\n",
				a.Bucket.Format("15:04"), a.Current, a.BaselineMean, a.Increase, a.ZScore)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func markdownIncident(w io.Writer, res *engine.Result) error {
	d := res.Diagnosis
	var b strings.Builder

	if d.RootCause == "Insufficient evidence" {
		b.WriteString("# Incident Report\n\n")
		b.WriteString("**Status:** Insufficient evidence\n\n")
		b.WriteString("The log evidence is not strong enough for a high-confidence diagnosis.\n")
	} else {
		b.WriteString("# Incident Report\n\n")
		fmt.Fprintf(&b, "**Root Cause:** %s\n\n", d.RootCause)
		fmt.Fprintf(&b, "**Confidence:** %.0f%%\n\n", d.Confidence*100)
		if d.Severity != "" {
			fmt.Fprintf(&b, "**Severity:** %s\n\n", d.Severity)
		}
	}

	if len(d.Evidence) > 0 {
		b.WriteString("## Evidence\n\n")
		b.WriteString("| Time | Type | Detail |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, ev := range d.Evidence {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", formatClock(ev.Timestamp), ev.Type, ev.Message)
		}
		b.WriteString("\n")
	}

	if len(d.Recommendations) > 0 {
		b.WriteString("## Recommended Actions\n\n")
		for i, r := range d.Recommendations {
			fmt.Fprintf(&b, "%d. %s\n", i+1, r)
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func markdownDiagnosis(b *strings.Builder, d *model.Diagnosis) {
	fmt.Fprintf(b, "- **Root Cause:** %s\n", d.RootCause)
	fmt.Fprintf(b, "- **Confidence:** %.2f\n", d.Confidence)
	if d.Severity != "" {
		fmt.Fprintf(b, "- **Severity:** %s\n", d.Severity)
	}
	if len(d.Evidence) > 0 {
		b.WriteString("\n**Evidence:**\n\n")
		for _, ev := range d.Evidence {
			fmt.Fprintf(b, "- %s `%s` — %s\n", formatClock(ev.Timestamp), ev.Type, ev.Message)
		}
	}
	if len(d.Recommendations) > 0 {
		b.WriteString("\n**Recommendations:**\n\n")
		for i, r := range d.Recommendations {
			fmt.Fprintf(b, "%d. %s\n", i+1, r)
		}
	}
}

// formatClock renders a time as HH:MM:SS, or "-" when zero.
func formatClock(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05")
}
