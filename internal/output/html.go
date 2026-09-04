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
	HasTimeline   bool
}

// renderHTMLTemplate is a fully self-contained report: inline CSS only, no
// external resources, so it works offline and as a CI artifact. The design is
// an "incident console": semantic severity colors, careful hierarchy, and a
// light/dark theme that follows the OS.
const renderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FaultLens Report</title>
<style>
  :root {
    color-scheme: light dark;
    --bg: #f4f5f7;
    --surface: #ffffff;
    --surface-2: #fafbfc;
    --border: #e2e4e8;
    --text: #1a1d21;
    --muted: #66707a;
    --faint: #9aa1a9;
    --accent: #0b6bcb;
    --accent-soft: #e5f1fb;
    --green: #1a7f37;
    --critical: #cf222e;
    --critical-soft: #ffebe9;
    --high: #bc4c00;
    --high-soft: #fff1e5;
    --medium: #9a6700;
    --medium-soft: #fff8c5;
    --mono: ui-monospace, SFMono-Regular, "Cascadia Mono", Consolas, Menlo, monospace;
  }
  @media (prefers-color-scheme: dark) {
    :root {
      --bg: #0d1117;
      --surface: #161b22;
      --surface-2: #1c2129;
      --border: #2b313a;
      --text: #e6edf3;
      --muted: #9aa4b0;
      --faint: #6e7681;
      --accent: #4d9fff;
      --accent-soft: #152c44;
      --green: #3fb950;
      --critical: #f85149;
      --critical-soft: #2d1517;
      --high: #ffa657;
      --high-soft: #2d1a0e;
      --medium: #d29922;
      --medium-soft: #2c2208;
    }
  }

  * { box-sizing: border-box; margin: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue",
                 Arial, sans-serif;
    background: var(--bg);
    color: var(--text);
    line-height: 1.55;
    -webkit-font-smoothing: antialiased;
    padding: 0 20px 60px;
  }
  .wrap { max-width: 940px; margin: 0 auto; }

  /* ---- header ---- */
  .header {
    padding: 34px 0 20px;
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .logo {
    width: 42px; height: 42px; border-radius: 11px;
    background: linear-gradient(135deg, var(--accent), #0a4a8a);
    display: grid; place-items: center;
    color: #fff; font-weight: 800; font-size: 15px; letter-spacing: .5px;
    box-shadow: 0 2px 8px rgba(11,107,203,.25);
    flex: none;
  }
  .header h1 { font-size: 20px; font-weight: 700; letter-spacing: -.2px; }
  .header .sub { color: var(--muted); font-size: 13px; }
  .header .tag { margin-left: auto; font-size: 11px; color: var(--faint);
    font-variant-numeric: tabular-nums; }

  /* ---- sections ---- */
  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 18px 20px;
    margin-top: 14px;
  }
  h2 {
    font-size: 12px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 1.1px; color: var(--faint);
    margin: 28px 0 -2px; display: flex; align-items: center; gap: 8px;
  }
  h2::after { content: ""; flex: 1; height: 1px; background: var(--border); }

  /* ---- summary stats ---- */
  .stats { display: grid; grid-template-columns: repeat(auto-fit,minmax(150px,1fr)); gap: 10px; }
  .stat {
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 14px 16px 12px;
  }
  .stat .num { font-size: 26px; font-weight: 750; letter-spacing: -.5px;
    font-variant-numeric: tabular-nums; line-height: 1.1; }
  .stat .lbl { font-size: 11px; text-transform: uppercase; letter-spacing: .8px;
    color: var(--muted); margin-top: 2px; }
  .stat.err .num { color: var(--critical); }
  .stat.warn .num { color: var(--medium); }
  .stat.fatal .num { color: var(--critical); }
  .meta { display: grid; grid-template-columns: auto 1fr; gap: 2px 14px;
    font-size: 13px; }
  .meta dt { color: var(--muted); }
  .meta dd { color: var(--text); word-break: break-all; }
  .meta dd code { font-family: var(--mono); font-size: 12px; }

  /* ---- timeline ---- */
  .legend { font-size: 12px; color: var(--muted); margin: 8px 0 0;
    display: flex; gap: 16px; flex-wrap: wrap; }
  .legend i { display: inline-block; width: 10px; height: 10px; border-radius: 2px;
    margin-right: 5px; vertical-align: -1px; }
  .legend .normal { background: var(--accent); }
  .legend .anomaly { background: var(--critical); }

  /* ---- diagnosis ---- */
  .dx { border-left: 4px solid var(--muted); padding: 18px 20px; }
  .sev-critical { border-color: var(--critical); }
  .sev-high { border-color: var(--high); }
  .sev-medium { border-color: var(--medium); }
  .sev-low { border-color: var(--green); }

  .dx .rc-label { font-size: 11px; text-transform: uppercase; letter-spacing: 1.2px;
    color: var(--muted); }
  .dx h3 { font-size: 22px; font-weight: 750; letter-spacing: -.4px; margin: 2px 0 8px; }
  .pill {
    display: inline-block; font-size: 11px; font-weight: 700;
    text-transform: uppercase; letter-spacing: .8px;
    padding: 3px 10px; border-radius: 999px;
  }
  .pill-critical { background: var(--critical-soft); color: var(--critical); }
  .pill-high { background: var(--high-soft); color: var(--high); }
  .pill-medium { background: var(--medium-soft); color: var(--medium); }
  .pill-low { background: var(--accent-soft); color: var(--accent); }
  .pill-insufficient { background: var(--medium-soft); color: var(--medium); }

  .conf { margin: 12px 0 4px; }
  .conf .conf-label { font-size: 12px; color: var(--muted); }
  .conf .conf-num { float: right; font-size: 12px; font-weight: 650;
    font-variant-numeric: tabular-nums; }
  .conf .bar { height: 7px; background: var(--surface-2);
    border: 1px solid var(--border); border-radius: 999px; overflow: hidden;
    margin-top: 4px; }
  .conf .bar > span { display: block; height: 100%;
    background: linear-gradient(90deg, var(--green), var(--accent));
    border-radius: 999px; }
  .sev-critical .bar > span { background: linear-gradient(90deg,#d16a6e,var(--critical)); }
  .sev-high .bar > span { background: linear-gradient(90deg,#d99b68,var(--high)); }

  .dx h4 { font-size: 12px; text-transform: uppercase; letter-spacing: .9px;
    color: var(--muted); margin: 16px 0 6px; }
  .ev-list { list-style: none; padding: 0; display: grid; gap: 6px; }
  .ev {
    background: var(--surface-2); border: 1px solid var(--border);
    border-radius: 8px; padding: 7px 11px; font-size: 13px;
    display: flex; gap: 10px; align-items: baseline;
  }
  .ev time { color: var(--faint); font-family: var(--mono); font-size: 12px;
    flex: none; font-variant-numeric: tabular-nums; }
  .badge { font-size: 10px; font-weight: 700; letter-spacing: .4px;
    padding: 1px 7px; border-radius: 5px; flex: none; text-transform: uppercase; }
  .b-error { background: var(--critical-soft); color: var(--critical); }
  .b-anomaly { background: var(--high-soft); color: var(--high); }
  .b-downstream { background: var(--accent-soft); color: var(--accent); }
  .b-temporal { background: var(--medium-soft); color: var(--medium); }
  .b-stack { background: var(--surface); color: var(--muted);
    border: 1px solid var(--border); }
  .ev p { color: var(--text); }

  ol.recs { padding-left: 0; list-style: none; display: grid; gap: 6px; }
  .recs li { display: flex; gap: 10px; align-items: baseline; font-size: 13px; }
  .recs .n { font-family: var(--mono); font-size: 12px; font-weight: 700;
    color: var(--accent); flex: none; }

  /* ---- tables ---- */
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th { font-size: 11px; text-transform: uppercase; letter-spacing: .7px;
    color: var(--muted); text-align: left; padding: 6px 10px;
    border-bottom: 1px solid var(--border); }
  td { padding: 8px 10px; border-bottom: 1px solid var(--border);
    vertical-align: top; }
  tbody tr:last-child td { border-bottom: none; }
  tbody tr:hover td { background: var(--surface-2); }
  td code, .msg { font-family: var(--mono); font-size: 12px; }
  .count { text-align: right; font-variant-numeric: tabular-nums; font-weight: 600; }
  .rank { color: var(--faint); width: 1%; white-space: nowrap; }

  .files { list-style: none; padding: 0; display: grid; gap: 4px; }
  .files code { font-family: var(--mono); font-size: 12px; color: var(--muted); }
  .empty { color: var(--muted); font-size: 13px; padding: 6px 2px; }
  ul.warns { margin: 0; padding-left: 18px; font-size: 13px; color: var(--high); }

  @media (max-width: 620px) {
    .header { flex-wrap: wrap; }
    .header .tag { margin-left: 0; width: 100%; }
  }
  @media print {
    body { background: #fff; color: #000; padding: 0; }
    .card { break-inside: avoid; }
  }
</style>
</head>
<body>
<div class="wrap">
  <header class="header">
    <div class="logo">FL</div>
    <div>
      <h1>FaultLens Report</h1>
      <div class="sub">See beyond the error.</div>
    </div>
    <div class="tag">{{.Summary.Format}} · {{.Summary.Source}}</div>
  </header>

  <!-- Summary -->
  <section class="card">
    <div class="stats">
      <div class="stat"><div class="num">{{.Summary.Events}}</div><div class="lbl">Events</div></div>
      <div class="stat err"><div class="num">{{.Summary.Errors}}</div><div class="lbl">Errors</div></div>
      <div class="stat warn"><div class="num">{{.Summary.Warnings}}</div><div class="lbl">Warnings</div></div>
      <div class="stat fatal"><div class="num">{{.Summary.Fatal}}</div><div class="lbl">Fatal</div></div>
    </div>
    <dl class="meta" style="margin-top:14px">
      {{if not .Summary.FirstEvent.IsZero}}
      <dt>Window</dt><dd>{{.Summary.FirstEvent.Format "2006-01-02 15:04:05"}} → {{.Summary.LastEvent.Format "2006-01-02 15:04:05"}}</dd>
      {{end}}
      <dt>Format</dt><dd>{{.Summary.Format}}</dd>
      <dt>Source</dt><dd><code>{{.Summary.Source}}</code></dd>
      {{if gt .Summary.ParsingWarnings 0}}
      <dt>⚠ Warnings</dt><dd>{{.Summary.ParsingWarnings}} lines could not be parsed</dd>
      {{end}}
    </dl>
  </section>

  <!-- Timeline -->
  {{if .TimelineSVG}}
  <h2>Error Timeline</h2>
  <section class="card">
    {{.TimelineSVG}}
    <div class="legend">
      <span><i class="normal"></i>normal bucket</span>
      <span><i class="anomaly"></i>anomaly</span>
    </div>
  </section>
  {{end}}

  <!-- Anomalies -->
  {{if .Anomalies}}
  <h2>Anomalies</h2>
  <section class="card">
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
  </section>
  {{end}}

  <!-- Diagnosis -->
  <h2>Diagnosis</h2>
  <section class="card dx sev-{{.Diagnosis.Severity}}">
    <div class="rc-label">Root Cause</div>
    <h3>{{.Diagnosis.RootCause}}</h3>
    <span class="pill pill-{{.Diagnosis.Severity}}">{{.Diagnosis.Severity}}</span>

    {{if ne .Diagnosis.RootCause "Insufficient evidence"}}
    <div class="conf">
      <span class="conf-label">Confidence</span>
      <span class="conf-num">{{.ConfidencePct}}%</span>
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
  <section class="card">
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
  </section>

  <!-- Source files -->
  {{if .SourceFiles}}
  <h2>Source Files</h2>
  <section class="card">
    <ul class="files">{{range .SourceFiles}}<li><code>{{.}}</code></li>{{end}}</ul>
  </section>
  {{end}}

  <!-- Config warnings -->
  {{if .ConfigWarnings}}
  <h2>Configuration Warnings</h2>
  <section class="card">
    <ul class="warns">{{range .ConfigWarnings}}<li>{{.}}</li>{{end}}</ul>
  </section>
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
		plotH  = 120
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

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" role="img" aria-label="Error timeline" style="width:100%%;height:auto">`, width, height)

	// Horizontal grid lines + baseline.
	for _, gy := range []int{0, plotH / 4, plotH / 2, plotH * 3 / 4, plotH} {
		op := "0.05"
		if gy == plotH {
			op = "0.25"
		}
		fmt.Fprintf(&b, `<line x1="0" y1="%d" x2="%d" y2="%d" stroke="currentColor" stroke-opacity="%s"/>`, gy, width, gy, op)
	}

	n := len(buckets)
	barW := float64(width) / float64(n)
	// Cap the bar inset for very wide charts.
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
		color := "#0b6bcb"
		opacity := "0.85"
		if anomAt[bucket.Start.Unix()] {
			color = "#cf222e"
			opacity = "1"
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="1" fill="%s" fill-opacity="%s"/>`,
			x, y, barW-gap, h, color, opacity)
	}
	// Time labels: first, middle, last.
	for _, idx := range []int{0, n / 2, n - 1} {
		anchor := "start"
		if idx == n-1 {
			anchor = "end"
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" font-size="10" fill="currentColor" fill-opacity="0.55" text-anchor="%s" font-family="ui-monospace,Consolas,monospace">%s</text>`,
			float64(idx)*barW+barW/2, height-6, anchor, buckets[idx].Start.Format("15:04"))
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
