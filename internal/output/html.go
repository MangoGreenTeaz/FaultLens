package output

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/MangoGreenTeaz/FaultLens/internal/anomaly"
	"github.com/MangoGreenTeaz/FaultLens/internal/engine"
	"github.com/MangoGreenTeaz/FaultLens/internal/timeline"
)

// maxTimelineBars caps how many buckets the SVG shows; older buckets are
// trimmed so reports stay readable for long-running processes.
const maxTimelineBars = 120

// htmlData wraps the result with precomputed presentation values.
type htmlData struct {
	*engine.Result
	TimelineSVG template.HTML
	SourceFiles []string
}

// renderHTMLTemplate is a fully self-contained report: inline CSS only, no
// external resources, so it works offline and as a CI artifact.
const renderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FaultLens Report</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; padding: 24px; background: #f6f8fa; color: #24292f; }
  .container { max-width: 960px; margin: 0 auto; }
  h1 { font-size: 1.5rem; margin: 0 0 4px; }
  h2 { font-size: 1.1rem; margin: 28px 0 8px; border-bottom: 1px solid #d0d7de; padding-bottom: 6px; }
  .muted { color: #57606a; }
  .card { background: #fff; border: 1px solid #d0d7de; border-radius: 8px; padding: 16px; margin: 8px 0; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; }
  .stat .value { font-size: 1.4rem; font-weight: 600; }
  .stat .label { color: #57606a; font-size: 0.8rem; }
  .severity-critical { color: #cf222e; font-weight: 600; }
  .severity-high { color: #bc4c00; font-weight: 600; }
  .severity-medium { color: #9a6700; font-weight: 600; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 8px; border-bottom: 1px solid #d0d7de; font-size: 0.9rem; }
  th { color: #57606a; font-weight: 500; }
  ul { margin: 4px 0; }
  code { background: #f0f1f3; padding: 2px 6px; border-radius: 4px; font-size: 0.85em; }
  @media (prefers-color-scheme: dark) {
    body { background: #0d1117; color: #e6edf3; }
    .card { background: #161b22; border-color: #30363d; }
    h2 { border-color: #30363d; }
    th, td { border-color: #30363d; }
    th, .label, .muted { color: #8b949e; }
    code { background: #21262d; }
  }
</style>
</head>
<body>
<div class="container">
  <h1>FaultLens Report</h1>
  <p class="muted">See beyond the error.</p>

  <h2>Summary</h2>
  <div class="grid">
    <div class="card stat"><div class="value">{{.Summary.Events}}</div><div class="label">Events</div></div>
    <div class="card stat"><div class="value">{{.Summary.Errors}}</div><div class="label">Errors</div></div>
    <div class="card stat"><div class="value">{{.Summary.Warnings}}</div><div class="label">Warnings</div></div>
    <div class="card stat"><div class="value">{{.Summary.Fatal}}</div><div class="label">Fatal</div></div>
  </div>
  <div class="card">
    <div><strong>Source:</strong> {{.Summary.Source}}</div>
    <div><strong>Format:</strong> {{.Summary.Format}}</div>
    {{if not .Summary.FirstEvent.IsZero}}<div><strong>Time Range:</strong> {{.Summary.FirstEvent.Format "2006-01-02 15:04:05"}} - {{.Summary.LastEvent.Format "2006-01-02 15:04:05"}}</div>{{end}}
    {{if gt .Summary.ParsingWarnings 0}}<div class="severity-medium"><strong>Parsing warnings:</strong> {{.Summary.ParsingWarnings}} lines could not be fully parsed</div>{{end}}
  </div>

  {{if .TimelineSVG}}
  <h2>Timeline</h2>
  <div class="card">{{.TimelineSVG}}</div>
  {{end}}

  {{if .Anomalies}}
  <h2>Anomalies</h2>
  <div class="card">
    <table>
      <tr><th>Time</th><th>Baseline</th><th>Current</th><th>Increase</th></tr>
      {{range .Anomalies}}<tr><td>{{.Bucket.Format "15:04"}}</td><td>{{printf "%.1f" .BaselineMean}}</td><td>{{.Current}}</td><td>{{printf "%.1fx" .Increase}}</td></tr>{{end}}
    </table>
  </div>
  {{end}}

  <h2>Diagnosis</h2>
  <div class="card">
    <div class="severity-{{.Diagnosis.Severity}}">Root Cause: {{.Diagnosis.RootCause}}</div>
    <div>Confidence: {{printf "%.2f" .Diagnosis.Confidence}}</div>
    <div>Severity: {{.Diagnosis.Severity}}</div>
    {{if .Diagnosis.Evidence}}
    <div style="margin-top:8px"><strong>Evidence:</strong></div>
    <ul>
      {{range .Diagnosis.Evidence}}<li>{{if not .Timestamp.IsZero}}{{.Timestamp.Format "15:04:05"}} — {{end}}<code>{{.Type}}</code> {{.Message}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Diagnosis.Recommendations}}
    <div style="margin-top:8px"><strong>Recommended:</strong></div>
    <ol>
      {{range .Diagnosis.Recommendations}}<li>{{.}}</li>{{end}}
    </ol>
    {{end}}
  </div>

  <h2>Error Groups</h2>
  {{if .ErrorGroups}}
  <div class="card">
    <table>
      <tr><th>#</th><th>Error</th><th>Occurrences</th><th>First seen</th></tr>
      {{range $i, $g := .ErrorGroups}}<tr><td>{{add $i 1}}</td><td><code>{{$g.Message}}</code></td><td>{{$g.Count}}</td><td>{{$g.FirstSeen.Format "15:04:05"}}</td></tr>{{end}}
    </table>
  </div>
  {{else}}<div class="card muted">No error groups</div>{{end}}

  {{if .SourceFiles}}
  <h2>Source Files</h2>
  <div class="card">
    <ul>{{range .SourceFiles}}<li><code>{{.}}</code></li>{{end}}</ul>
  </div>
  {{end}}

  {{if .ConfigWarnings}}
  <h2>Configuration Warnings</h2>
  <div class="card">
    <ul>{{range .ConfigWarnings}}<li>{{.}}</li>{{end}}</ul>
  </div>
  {{end}}
</div>
</body>
</html>
`

// RenderHTML writes a fully offline, single-file HTML report. It is suitable
// for CI artifacts, incident sharing and postmortems.
func RenderHTML(w io.Writer, res *engine.Result) error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(renderHTMLTemplate)
	if err != nil {
		return err
	}

	data := htmlData{
		Result:      res,
		TimelineSVG: timelineSVG(res.Timeline, res.Anomalies),
		SourceFiles: sourceFiles(res),
	}
	return tmpl.Execute(w, data)
}

// timelineSVG renders the per-minute error counts as an inline SVG bar chart.
// Buckets flagged by the anomaly detector are drawn in red. The chart is
// capped at maxTimelineBars buckets to stay readable.
func timelineSVG(tl []timeline.Bucket, anoms []anomaly.Detection) template.HTML {
	if len(tl) == 0 {
		return ""
	}
	start := 0
	if len(tl) > maxTimelineBars {
		start = len(tl) - maxTimelineBars
	}
	buckets := tl[start:]

	const (
		width  = 800
		plotH  = 100
		labelH = 20
	)
	height := plotH + labelH

	maxErr := 1
	for _, b := range buckets {
		if b.Errors > maxErr {
			maxErr = b.Errors
		}
	}
	anomAt := make(map[int64]bool, len(anoms))
	for _, a := range anoms {
		anomAt[a.Bucket.Unix()] = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" role="img" aria-label="Error timeline">`, width, height)
	fmt.Fprintf(&b, `<line x1="0" y1="%d" x2="%d" y2="%d" stroke="#d0d7de"/>`, plotH, width, plotH)

	n := len(buckets)
	barW := float64(width) / float64(n)
	for i, bucket := range buckets {
		x := float64(i) * barW
		h := float64(bucket.Errors) / float64(maxErr) * plotH
		if h < 1 {
			h = 1
		}
		y := plotH - h
		color := "#0969da"
		if anomAt[bucket.Start.Unix()] {
			color = "#cf222e"
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.2f" height="%.1f" fill="%s"/>`, x, y, barW-0.5, h, color)
	}
	// Time labels: first, middle, last.
	for _, idx := range []int{0, n / 2, n - 1} {
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" font-size="10" fill="#8b949e">%s</text>`,
			float64(idx)*barW, height-4, buckets[idx].Start.Format("15:04"))
	}
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

// sourceFiles returns the sorted, unique set of files that produced error
// group examples (empty for stdin or when no error groups exist).
func sourceFiles(res *engine.Result) []string {
	set := make(map[string]bool)
	for i := range res.ErrorGroups {
		for _, ex := range res.ErrorGroups[i].Examples {
			if ex.Source != "" {
				set[ex.Source] = true
			}
		}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
