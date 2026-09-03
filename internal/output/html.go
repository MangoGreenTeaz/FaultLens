package output

import (
	"html/template"
	"io"

	"github.com/faultlens/faultlens/internal/engine"
)

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
	return tmpl.Execute(w, res)
}
