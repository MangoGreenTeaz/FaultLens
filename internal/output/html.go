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
    color-scheme: dark;
    --bg: #0B0F19;
    --surface: #151B2B;
    --surface-2: #1B2334;
    --surface-3: #202A3E;
    --border: rgba(255,255,255,0.08);
    --border-strong: rgba(255,255,255,0.14);
    --text: #F9FAFB;
    --muted: #9CA3AF;
    --faint: #6B7280;
    --primary: #6366F1;
    --primary-soft: rgba(99,102,241,0.14);
    --success: #22C55E;
    --warning: #F59E0B;
    --danger: #EF4444;
    --danger-soft: rgba(239,68,68,0.14);
    --warning-soft: rgba(245,158,11,0.14);
    --sans: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif;
    --mono: "JetBrains Mono", ui-monospace, SFMono-Regular, "Cascadia Mono", Consolas, Menlo, monospace;
  }
  * { box-sizing: border-box; margin: 0; }
  html { background: var(--bg); }
  body {
    font-family: var(--sans);
    background:
      radial-gradient(1200px 500px at 80% -10%, rgba(99,102,241,0.08), transparent 60%),
      var(--bg);
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
		width  = 880
		plotH  = 110
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
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" role="img" aria-label="Error timeline" style="width:100%%;height:auto">`, width, height)

	// Grid lines + baseline (subtle on the dark background).
	for _, gy := range []int{0, plotH / 4, plotH / 2, plotH * 3 / 4, plotH} {
		op := "0.06"
		if gy == plotH {
			op = "0.18"
		}
		fmt.Fprintf(&b, `<line x1="0" y1="%d" x2="%d" y2="%d" stroke="#ffffff" stroke-opacity="%s"/>`, gy, width, gy, op)
	}

	n := len(buckets)
	barW := float64(width) / float64(n)
	gap := barW * 0.22
	if gap > 2.5 {
		gap = 2.5
	}
	for i, bucket := range buckets {
		x := float64(i)*barW + gap/2
		h := float64(bucket.Errors) / float64(maxErr) * (plotH - 4)
		if h < 1 {
			h = 1
		}
		y := plotH - h
		color := "#6366F1"
		opacity := "0.9"
		if anomAt[bucket.Start.Unix()] {
			color = "#EF4444"
			opacity = "1"
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="1.5" fill="%s" fill-opacity="%s"/>`,
			x, y, barW-gap, h, color, opacity)
	}
	for _, idx := range []int{0, n / 2, n - 1} {
		anchor := "start"
		if idx == n-1 {
			anchor = "end"
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" font-size="10" fill="#6B7280" text-anchor="%s" font-family="ui-monospace,Consolas,monospace">%s</text>`,
			float64(idx)*barW+barW/2, height-5, anchor, buckets[idx].Start.Format("15:04"))
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
