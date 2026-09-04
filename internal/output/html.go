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
	TimelineSVG   template.HTML
	SourceFiles   []string
	ConfidencePct int
}

// renderHTMLTemplate is a fully self-contained offline report (no CDN, no
// external JS). The design follows a modern developer-tool console aesthetic:
// dark-first, high information density, semantic severity colors, no
// gradients.
const renderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FaultLens Report</title>
<style>
  :root {
    color-scheme: light;
    --bg: #F7F8FA;
    --surface: #FFFFFF;
    --surface-2: #F3F4F6;
    --surface-3: #E9EBEF;
    --border: rgba(17,24,39,0.09);
    --border-strong: rgba(17,24,39,0.18);
    --text: #1F2328;
    --muted: #6B7280;
    --faint: #9CA3AF;
    --primary: #4F46E5;
    --primary-soft: rgba(79,70,229,0.10);
    --success: #16A34A;
    --warning: #D97706;
    --danger: #DC2626;
    --danger-soft: rgba(220,38,38,0.10);
    --warning-soft: rgba(217,119,6,0.12);
    --sans: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif;
    --mono: "JetBrains Mono", ui-monospace, SFMono-Regular, "Cascadia Mono", Consolas, Menlo, monospace;
  }
  * { box-sizing: border-box; margin: 0; }
  html { background: var(--bg); }
  body {
    font-family: var(--sans);
    background: var(--bg);
    color: var(--text);
    line-height: 1.55;
    -webkit-font-smoothing: antialiased;
    padding: 0 24px 64px;
  }
  .wrap { max-width: 1080px; margin: 0 auto; }

  /* ---- header ---- */
  .header {
    display: flex; align-items: center; gap: 12px;
    padding: 22px 0 16px;
    border-bottom: 1px solid var(--border);
    margin-bottom: 20px;
  }
  .logo {
    width: 32px; height: 32px; border-radius: 8px;
    background: var(--primary);
    display: grid; place-items: center;
    font-family: var(--mono); font-weight: 700; font-size: 12px;
    color: #fff; letter-spacing: .5px; flex: none;
  }
  .header h1 { font-size: 15px; font-weight: 700; letter-spacing: -.2px; }
  .header .sub { color: var(--faint); font-size: 12px; }
  .header .crumb { display: flex; align-items: baseline; gap: 6px; }
  .header .crumb b { color: var(--muted); font-weight: 600; }
  .header .tag { margin-left: auto; font-family: var(--mono); font-size: 11px;
    color: var(--faint); }

  /* ---- section titles ---- */
  h2 {
    font-size: 11px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 1.2px; color: var(--faint);
    margin: 26px 0 8px;
  }

  /* ---- metric cards ---- */
  .metrics { display: grid; grid-template-columns: repeat(auto-fit,minmax(190px,1fr)); gap: 10px; }
  .metric {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 14px 16px 12px;
    transition: border-color .15s ease;
  }
  .metric:hover { border-color: var(--border-strong); }
  .metric .lbl { font-size: 11px; text-transform: uppercase; letter-spacing: .9px;
    color: var(--faint); }
  .metric .num { font-size: 26px; font-weight: 700; letter-spacing: -.5px;
    margin-top: 4px; font-variant-numeric: tabular-nums; line-height: 1.1; }
  .metric .dot { display: inline-block; width: 8px; height: 8px; border-radius: 2px;
    margin-right: 6px; vertical-align: 2px; }
  .dot-critical { background: var(--danger); }
  .dot-warning { background: var(--warning); }
  .dot-success { background: var(--success); }
  .dot-primary { background: var(--primary); }

  .meta { display: grid; grid-template-columns: auto 1fr; gap: 3px 16px;
    font-size: 12.5px; margin-top: 14px; padding-top: 12px;
    border-top: 1px solid var(--border); }
  .meta dt { color: var(--faint); }
  .meta dd { color: var(--text); word-break: break-all; }
  .meta dd code { font-family: var(--mono); font-size: 11.5px; }

  /* ---- timeline ---- */
  .panel { background: var(--surface); border: 1px solid var(--border);
    border-radius: 12px; padding: 16px; }
  .legend { display: flex; gap: 16px; margin-top: 10px;
    font-size: 11.5px; color: var(--muted); }
  .legend i { display: inline-block; width: 9px; height: 9px; border-radius: 2px;
    margin-right: 6px; vertical-align: -1px; }
  .legend .normal { background: var(--primary); }
  .legend .anomaly { background: var(--danger); }

  /* ---- diagnosis ---- */
  .dx { border-left: 3px solid var(--border-strong); border-radius: 12px;
    background: var(--surface); border: 1px solid var(--border); border-left-width: 3px;
    padding: 18px 20px; }
  .sev-critical { border-left-color: var(--danger); }
  .sev-high { border-left-color: var(--warning); }
  .sev-medium { border-left-color: var(--warning); }
  .sev-low { border-left-color: var(--success); }

  .dx .rc-label { font-size: 11px; text-transform: uppercase; letter-spacing: 1.2px;
    color: var(--faint); }
  .dx h3 { font-size: 20px; font-weight: 700; letter-spacing: -.3px; margin: 3px 0 10px; }
  .pill { display: inline-block; font-size: 10px; font-weight: 700;
    text-transform: uppercase; letter-spacing: .9px; padding: 3px 10px;
    border-radius: 6px; }
  .pill-critical { background: var(--danger-soft); color: var(--danger); }
  .pill-high { background: var(--warning-soft); color: var(--warning); }
  .pill-medium { background: var(--warning-soft); color: var(--warning); }
  .pill-low { background: rgba(34,197,94,.14); color: var(--success); }
  .pill-insufficient { background: var(--warning-soft); color: var(--warning); }

  .conf { margin: 14px 0 2px; }
  .conf .row { display: flex; justify-content: space-between; font-size: 12px; }
  .conf .conf-label { color: var(--muted); }
  .conf .conf-num { color: var(--text); font-weight: 650;
    font-variant-numeric: tabular-nums; }
  .conf .bar { height: 5px; background: var(--surface-3); border-radius: 999px;
    overflow: hidden; margin-top: 5px; }
  .conf .bar > span { display: block; height: 100%; border-radius: 999px;
    background: var(--primary); }
  .sev-critical .bar > span { background: var(--danger); }
  .sev-high .bar > span { background: var(--warning); }

  .dx h4 { font-size: 11px; text-transform: uppercase; letter-spacing: 1px;
    color: var(--faint); margin: 18px 0 8px; }

  .ev-list { list-style: none; padding: 0; display: grid; gap: 5px; }
  .ev {
    display: flex; gap: 12px; align-items: baseline;
    padding: 8px 12px; border-radius: 8px;
    background: var(--surface-2);
    border: 1px solid var(--border);
    font-size: 12.5px;
  }
  .ev:hover { border-color: var(--border-strong); }
  .ev time { color: var(--faint); font-family: var(--mono); font-size: 11.5px;
    flex: none; font-variant-numeric: tabular-nums; }
  .badge { font-size: 9.5px; font-weight: 700; letter-spacing: .5px;
    padding: 1px 7px; border-radius: 4px; flex: none; text-transform: uppercase;
    font-family: var(--mono); }
  .b-error { background: var(--danger-soft); color: var(--danger); }
  .b-anomaly { background: var(--warning-soft); color: var(--warning); }
  .b-downstream { background: var(--primary-soft); color: var(--primary); }
  .b-temporal { background: var(--warning-soft); color: var(--warning); }
  .b-stack { background: var(--surface-3); color: var(--muted); }
  .ev p { color: var(--text); }

  ol.recs { list-style: none; padding: 0; display: grid; gap: 6px; }
  .recs li { display: flex; gap: 10px; align-items: baseline; font-size: 13px; }
  .recs .n { font-family: var(--mono); font-size: 11.5px; font-weight: 700;
    color: var(--primary); flex: none; }

  /* ---- tables (no vertical lines, compact, hover rows) ---- */
  table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
  thead th { font-size: 10px; text-transform: uppercase; letter-spacing: .8px;
    color: var(--faint); text-align: left; padding: 6px 10px;
    border-bottom: 1px solid var(--border); font-weight: 600; }
  tbody td { padding: 8px 10px; border-bottom: 1px solid var(--border);
    vertical-align: middle; }
  tbody tr { transition: background .12s ease; }
  tbody tr:hover { background: var(--surface-2); }
  tbody tr:last-child td { border-bottom: none; }
  tbody tr:hover td:first-child { border-radius: 8px 0 0 8px; }
  tbody tr:hover td:last-child { border-radius: 0 8px 8px 0; }
  td code, .msg { font-family: var(--mono); font-size: 11.5px; }
  .count { text-align: right; font-variant-numeric: tabular-nums; font-weight: 600; }
  .rank { color: var(--faint); width: 1%; white-space: nowrap; font-family: var(--mono); font-size: 11px; }

  .files { list-style: none; padding: 0; display: grid; gap: 5px; }
  .files code { font-family: var(--mono); font-size: 12px; color: var(--muted); }
  .empty { color: var(--muted); font-size: 13px; padding: 4px 2px; }
  ul.warns { margin: 0; padding-left: 18px; font-size: 12.5px; color: var(--warning); }

  @media (max-width: 640px) { .header .tag { display: none; } }
  @media print {
    body { background: #fff; color: #000; }
    .panel, .metric, .dx, .ev { background: #fff; }
  }
</style>
</head>
<body>
<div class="wrap">
  <header class="header">
    <div class="logo">FL</div>
    <div class="crumb">
      <h1>FaultLens</h1>
      <span style="color:var(--faint)">/</span>
      <b>{{.Summary.Source}}</b>
    </div>
    <div class="tag">{{.Summary.Format}}</div>
  </header>

  <!-- Summary -->
  <div class="metrics">
    <div class="metric"><div class="lbl">Events</div><div class="num">{{.Summary.Events}}</div></div>
    <div class="metric"><div class="lbl"><span class="dot dot-critical"></span>Errors</div><div class="num">{{.Summary.Errors}}</div></div>
    <div class="metric"><div class="lbl"><span class="dot dot-warning"></span>Warnings</div><div class="num">{{.Summary.Warnings}}</div></div>
    <div class="metric"><div class="lbl"><span class="dot dot-critical"></span>Fatal</div><div class="num">{{.Summary.Fatal}}</div></div>
  </div>
  <dl class="meta">
    {{if not .Summary.FirstEvent.IsZero}}
    <dt>Window</dt><dd>{{.Summary.FirstEvent.Format "2006-01-02 15:04:05"}} → {{.Summary.LastEvent.Format "2006-01-02 15:04:05"}}</dd>
    {{end}}
    <dt>Format</dt><dd>{{.Summary.Format}}</dd>
    <dt>Source</dt><dd><code>{{.Summary.Source}}</code></dd>
    {{if gt .Summary.ParsingWarnings 0}}
    <dt>Warnings</dt><dd>{{.Summary.ParsingWarnings}} lines could not be parsed</dd>
    {{end}}
  </dl>

  <!-- Timeline -->
  {{if .TimelineSVG}}
  <h2>Error Timeline</h2>
  <div class="panel">
    {{.TimelineSVG}}
    <div class="legend">
      <span><i class="normal"></i>normal</span>
      <span><i class="anomaly"></i>anomaly</span>
    </div>
  </div>
  {{end}}

  <!-- Anomalies -->
  {{if .Anomalies}}
  <h2>Anomalies</h2>
  <div class="panel">
    <table>
      <thead><tr><th>Time</th><th>Baseline</th><th>Errors</th><th>Increase</th></tr></thead>
      <tbody>
        {{range .Anomalies}}
        <tr><td><code>{{.Bucket.Format "2006-01-02 15:04"}}</code></td>
            <td>{{printf "%.1f" .BaselineMean}}</td>
            <td>{{.Current}}</td>
            <td class="count">{{printf "%.1fx" .Increase}}</td></tr>
        {{end}}
      </tbody>
    </table>
  </div>
  {{end}}

  <!-- Diagnosis -->
  <h2>Diagnosis</h2>
  <section class="dx sev-{{.Diagnosis.Severity}}">
    <div class="rc-label">Root Cause</div>
    <h3>{{.Diagnosis.RootCause}}</h3>
    <span class="pill pill-{{.Diagnosis.Severity}}">{{.Diagnosis.Severity}}</span>

    {{if ne .Diagnosis.RootCause "Insufficient evidence"}}
    <div class="conf">
      <div class="row"><span class="conf-label">Confidence</span><span class="conf-num">{{.ConfidencePct}}%</span></div>
      <div class="bar"><span style="width:{{.ConfidencePct}}%"></span></div>
    </div>
    {{end}}

    {{if .Diagnosis.Evidence}}
    <h4>Evidence</h4>
    <ul class="ev-list">
      {{range .Diagnosis.Evidence}}
      <li class="ev">
        {{if not .Timestamp.IsZero}}<time>{{.Timestamp.Format "15:04:05"}}</time>{{end}}
        <span class="badge b-{{badgeClass .Type}}">{{.Type}}</span>
        <p>{{.Message}}</p>
      </li>
      {{end}}
    </ul>
    {{end}}

    {{if .Diagnosis.Recommendations}}
    <h4>Recommended Actions</h4>
    <ol class="recs">
      {{range $i, $r := .Diagnosis.Recommendations}}
      <li><span class="n">{{add $i 1}}</span><span>{{$r}}</span></li>
      {{end}}
    </ol>
    {{end}}
  </section>

  <!-- Error groups -->
  <h2>Error Groups</h2>
  <div class="panel">
    {{if .ErrorGroups}}
    <table>
      <thead><tr><th class="rank">#</th><th>Error</th><th class="count">Occurrences</th><th>First seen</th></tr></thead>
      <tbody>
        {{range $i, $g := .ErrorGroups}}
        <tr><td class="rank">{{add $i 1}}</td>
            <td><code>{{$g.Message}}</code></td>
            <td class="count">{{$g.Count}}</td>
            <td><code>{{if not $g.FirstSeen.IsZero}}{{$g.FirstSeen.Format "15:04:05"}}{{end}}</code></td></tr>
        {{end}}
      </tbody>
    </table>
    {{else}}<div class="empty">No error groups</div>{{end}}
  </div>

  <!-- Source files -->
  {{if .SourceFiles}}
  <h2>Source Files</h2>
  <div class="panel">
    <ul class="files">{{range .SourceFiles}}<li><code>{{.}}</code></li>{{end}}</ul>
  </div>
  {{end}}

  <!-- Config warnings -->
  {{if .ConfigWarnings}}
  <h2>Configuration Warnings</h2>
  <div class="panel">
    <ul class="warns">{{range .ConfigWarnings}}<li>{{.}}</li>{{end}}</ul>
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
		"add":        func(a, b int) int { return a + b },
		"badgeClass": evidenceBadgeClass,
	}).Parse(renderHTMLTemplate)
	if err != nil {
		return err
	}

	data := htmlData{
		Result:        res,
		TimelineSVG:   timelineSVG(res.Timeline, res.Anomalies),
		SourceFiles:   sourceFiles(res),
		ConfidencePct: int(res.Diagnosis.Confidence*100 + 0.5),
	}
	return tmpl.Execute(w, data)
}

// evidenceBadgeClass maps an evidence type to a CSS badge modifier.
func evidenceBadgeClass(t string) string {
	switch t {
	case "ERROR_PATTERN":
		return "error"
	case "ANOMALY":
		return "anomaly"
	case "DOWNSTREAM_IMPACT":
		return "downstream"
	case "TIMELINE_CORRELATION":
		return "temporal"
	case "STACK_TRACE":
		return "stack"
	default:
		return "stack"
	}
}

// timelineSVG renders the per-minute error counts as an area chart (fill +
// line), with anomalous buckets marked in red — a Datadog/Grafana-style
// metric view. The chart is capped at maxTimelineBars buckets.
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
		width  = 880
		plotH  = 130
		labelH = 22
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

	n := len(buckets)
	barW := float64(width) / float64(n)

	type pt struct{ x, y float64 }
	pts := make([]pt, n)
	for i, bucket := range buckets {
		x := (float64(i) + 0.5) * barW
		h := float64(bucket.Errors) / float64(maxErr) * (plotH - 12)
		pts[i] = pt{x: x, y: plotH - h}
	}
	base := float64(plotH)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" role="img" aria-label="Error timeline" style="width:100%%;height:auto">`, width, height)

	// Horizontal grid lines with y-axis labels (max / half / zero).
	for _, frac := range []float64{1, 0.5, 0} {
		y := plotH - frac*(plotH-12)
		op := "0.06"
		if frac == 0 {
			op = "0.15"
		}
		fmt.Fprintf(&b, `<line x1="0" y1="%.1f" x2="%d" y2="%.1f" stroke="#1F2328" stroke-opacity="%s"/>`, y, width, y, op)
		val := int(float64(maxErr)*frac + 0.5)
		fmt.Fprintf(&b, `<text x="4" y="%.1f" font-size="9" fill="#6B7280" font-family="ui-monospace,Consolas,monospace">%d</text>`, y-3, val)
	}

	// Area fill under the line (solid translucent primary, no gradient).
	var area strings.Builder
	area.WriteString("M")
	for i, p := range pts {
		fmt.Fprintf(&area, " %.1f,%.1f", p.x, p.y)
		if i < n-1 {
			area.WriteString(" L")
		}
	}
	fmt.Fprintf(&area, " L%.1f,%.1f L%.1f,%.1f Z", pts[n-1].x, base, pts[0].x, base)
	fmt.Fprintf(&b, `<path d="%s" fill="#4F46E5" fill-opacity="0.12"/>`, area.String())

	// Line connecting the points.
	var line strings.Builder
	line.WriteString("M")
	for i, p := range pts {
		fmt.Fprintf(&line, " %.1f,%.1f", p.x, p.y)
		if i < n-1 {
			line.WriteString(" L")
		}
	}
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#4F46E5" stroke-width="2" stroke-linejoin="round" stroke-linecap="round"/>`, line.String())

	// Anomalous buckets: red markers on the line.
	for i, p := range pts {
		if anomAt[buckets[i].Start.Unix()] {
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="3.5" fill="#DC2626"/>`, p.x, p.y)
		}
	}

	// Time labels: first, middle, last.
	for _, idx := range []int{0, n / 2, n - 1} {
		anchor := "middle"
		if idx == 0 {
			anchor = "start"
		}
		if idx == n-1 {
			anchor = "end"
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" font-size="10" fill="#6B7280" text-anchor="%s" font-family="ui-monospace,Consolas,monospace">%s</text>`,
			pts[idx].x, height-5, anchor, buckets[idx].Start.Format("15:04"))
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
