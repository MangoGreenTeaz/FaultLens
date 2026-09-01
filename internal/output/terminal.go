// Package output renders an engine.Result as terminal text, JSON or
// Markdown.
package output

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/faultlens/faultlens/internal/engine"
	"github.com/faultlens/faultlens/internal/model"
)

const sepLine = "────────────────────────────────────────────"

// RenderTerminal writes the human-readable report for the requested view:
// "report", "errors", "timeline" or "incident".
func RenderTerminal(w io.Writer, res *engine.Result, kind string) error {
	switch kind {
	case "errors":
		return renderTerminalErrors(w, res)
	case "timeline":
		return renderTerminalTimeline(w, res)
	case "incident":
		return renderTerminalIncident(w, res)
	default:
		return renderTerminalReport(w, res)
	}
}

// renderTerminalReport is the default full report.
func renderTerminalReport(w io.Writer, res *engine.Result) error {
	var b strings.Builder
	b.WriteString("FaultLens\n")
	b.WriteString(sepLine + "\n\n")

	b.WriteString("Log Summary\n\n")
	fmt.Fprintf(&b, "Events:       %s\n", comma(res.Summary.Events))
	fmt.Fprintf(&b, "Errors:       %s\n", comma(res.Summary.Errors))
	fmt.Fprintf(&b, "Warnings:     %s\n", comma(res.Summary.Warnings))
	if !res.Summary.FirstEvent.IsZero() {
		fmt.Fprintf(&b, "Time Range:   %s - %s\n",
			res.Summary.FirstEvent.Format("15:04:05"), res.Summary.LastEvent.Format("15:04:05"))
	}
	fmt.Fprintf(&b, "Format:       %s\n", res.Summary.Format)
	if res.Summary.ParsingWarnings > 0 {
		fmt.Fprintf(&b, "Parsing warnings: %d lines could not be fully parsed\n", res.Summary.ParsingWarnings)
	}
	b.WriteString("\n" + sepLine + "\n\n")

	b.WriteString("Anomalies\n\n")
	if len(res.Anomalies) == 0 {
		b.WriteString("None detected\n")
	} else {
		for _, a := range res.Anomalies {
			fmt.Fprintf(&b, "%s\nError rate increased %.1fx (baseline %.1f)\n\n",
				a.Bucket.Format("15:04"), a.Increase, a.BaselineMean)
		}
	}
	b.WriteString(sepLine + "\n\n")

	b.WriteString("Top Error Groups\n\n")
	if len(res.ErrorGroups) == 0 {
		b.WriteString("No error groups\n")
	} else {
		for i := 0; i < min(5, len(res.ErrorGroups)); i++ {
			g := res.ErrorGroups[i]
			fmt.Fprintf(&b, "%d. %s\n", i+1, g.Message)
			fmt.Fprintf(&b, "   %s occurrences\n\n", comma(g.Count))
		}
	}
	b.WriteString(sepLine + "\n\n")

	b.WriteString("Diagnosis\n\n")
	writeDiagnosis(&b, res.Diagnosis)

	_, err := io.WriteString(w, b.String())
	return err
}

// renderTerminalErrors prints the error-group ranking.
func renderTerminalErrors(w io.Writer, res *engine.Result) error {
	var b strings.Builder
	b.WriteString("ERROR GROUPS\n\n")
	if len(res.ErrorGroups) == 0 {
		b.WriteString("No error groups\n")
	} else {
		for i, g := range res.ErrorGroups {
			fmt.Fprintf(&b, "%d. %s\n", i+1, g.Message)
			fmt.Fprintf(&b, "   Occurrences: %s\n", comma(g.Count))
			if !g.FirstSeen.IsZero() {
				fmt.Fprintf(&b, "   First seen:  %s\n", g.FirstSeen.Format("15:04:05"))
				fmt.Fprintf(&b, "   Last seen:   %s\n", g.LastSeen.Format("15:04:05"))
			}
			if len(g.Examples) > 0 {
				fmt.Fprintf(&b, "   Example:     %s\n", g.Examples[0].Raw)
			}
			b.WriteString("\n")
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// renderTerminalTimeline prints the per-minute buckets plus any anomalies.
func renderTerminalTimeline(w io.Writer, res *engine.Result) error {
	var b strings.Builder
	b.WriteString("TIMELINE\n\n")
	if len(res.Timeline) == 0 {
		b.WriteString("No timeline data\n")
	} else {
		for _, tb := range res.Timeline {
			fmt.Fprintf(&b, "%s  total=%-6d errors=%-6d warnings=%-6d fatal=%d\n",
				tb.Start.Format("15:04"), tb.Total, tb.Errors, tb.Warnings, tb.Fatal)
		}
	}
	if len(res.Anomalies) > 0 {
		b.WriteString("\nANOMALY DETECTED\n")
		for _, a := range res.Anomalies {
			fmt.Fprintf(&b, "\nTime:\n%s\n\nBaseline error count:\n%.1f\n\nCurrent error count:\n%d\n\nIncrease:\n%.1fx\n\n",
				a.Bucket.Format("15:04"), a.BaselineMean, a.Current, a.Increase)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// renderTerminalIncident prints the incident-oriented diagnosis.
func renderTerminalIncident(w io.Writer, res *engine.Result) error {
	d := res.Diagnosis
	var b strings.Builder

	if d.RootCause == "Insufficient evidence" {
		b.WriteString("Insufficient evidence\n\n")
		b.WriteString("The log evidence is not strong enough for a high-confidence diagnosis.\n\n")
		if len(d.Evidence) > 0 {
			b.WriteString("Possible signals:\n")
			for _, ev := range d.Evidence {
				fmt.Fprintf(&b, "- %s\n", ev.Message)
			}
		}
		b.WriteString("\n")
	} else {
		b.WriteString("Incident detected\n\n")
		fmt.Fprintf(&b, "Root Cause:\n%s\n\n", d.RootCause)
		fmt.Fprintf(&b, "Confidence:\n%.0f%%\n\n", d.Confidence*100)
		if d.Severity != "" {
			fmt.Fprintf(&b, "Severity:\n%s\n\n", d.Severity)
		}
	}

	if len(d.Evidence) > 0 {
		b.WriteString("Evidence:\n")
		for _, ev := range d.Evidence {
			ts := ""
			if !ev.Timestamp.IsZero() {
				ts = ev.Timestamp.Format("15:04:05") + "  "
			}
			fmt.Fprintf(&b, "%s%s\n", ts, ev.Message)
		}
		b.WriteString("\n")
	}

	if len(d.Recommendations) > 0 {
		b.WriteString("Recommended:\n")
		for i, r := range d.Recommendations {
			fmt.Fprintf(&b, "%d. %s\n", i+1, r)
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// writeDiagnosis renders the diagnosis block shared by terminal report views.
func writeDiagnosis(b *strings.Builder, d *model.Diagnosis) {
	fmt.Fprintf(b, "ROOT CAUSE\n%s\n\n", d.RootCause)
	fmt.Fprintf(b, "Confidence\n%.2f\n\n", d.Confidence)
	if d.Severity != "" {
		fmt.Fprintf(b, "Severity\n%s\n\n", d.Severity)
	}
	if len(d.Evidence) > 0 {
		b.WriteString("Evidence\n")
		for _, ev := range d.Evidence {
			ts := ""
			if !ev.Timestamp.IsZero() {
				ts = ev.Timestamp.Format("15:04:05") + "  "
			}
			fmt.Fprintf(b, "%s%s\n%s\n\n", ts, ev.Type, ev.Message)
		}
	}
	if len(d.Recommendations) > 0 {
		b.WriteString("Recommendations\n")
		for i, r := range d.Recommendations {
			fmt.Fprintf(b, "%d. %s\n", i+1, r)
		}
		b.WriteString("\n")
	}
}

// comma renders an integer with thousands separators.
func comma(n int) string {
	s := strconv.Itoa(n)
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}
