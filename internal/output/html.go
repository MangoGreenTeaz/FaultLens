package output

import (
	"fmt"
	"html/template"
	"io"
	"math"
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
// external JS). The design follows a modern card-based light console aesthetic:
// crisp white elevated surfaces, subtle shadows, semantic badges, and high
// information density.
const renderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FaultLens Report</title>
<style>
  :root {
    color-scheme: light;
    --bg: #f8fafc;
    --bg-mesh: radial-gradient(ellipse at 50% -10%, rgba(79, 70, 229, 0.07) 0%, transparent 60%);
    --surface: #ffffff;
    --surface-card: #ffffff;
    --surface-subtle: #f1f5f9;
    --surface-code: #f8fafc;
    --border: #e2e8f0;
    --border-strong: #cbd5e1;
    --border-glow: rgba(79, 70, 229, 0.25);
    --text: #0f172a;
    --muted: #475569;
    --faint: #64748b;
    --primary: #4f46e5;
    --primary-light: #6366f1;
    --primary-soft: #eef2ff;
    --primary-border: #c7d2fe;
    --primary-glow: rgba(79, 70, 229, 0.15);
    --danger: #e11d48;
    --danger-light: #f43f5e;
    --danger-soft: #ffe4e6;
    --danger-border: #fecdd3;
    --danger-glow: rgba(225, 29, 72, 0.15);
    --warning: #d97706;
    --warning-soft: #fef3c7;
    --warning-border: #fde68a;
    --warning-glow: rgba(217, 119, 6, 0.15);
    --success: #059669;
    --success-soft: #d1fae5;
    --success-border: #a7f3d0;
    --success-glow: rgba(5, 150, 105, 0.15);
    --info: #0284c7;
    --info-soft: #e0f2fe;
    --info-border: #bae6fd;
    --sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Inter", "Helvetica Neue", Arial, sans-serif;
    --mono: ui-monospace, SFMono-Regular, "Cascadia Code", "JetBrains Mono", Consolas, Menlo, monospace;
    --radius-lg: 16px;
    --radius-md: 12px;
    --radius-sm: 8px;
    --radius-pill: 9999px;
  }

  * { box-sizing: border-box; margin: 0; padding: 0; }
  html { background: var(--bg); color: var(--text); }
  body {
    font-family: var(--sans);
    background: var(--bg);
    background-image: var(--bg-mesh);
    background-repeat: no-repeat;
    background-attachment: fixed;
    color: var(--text);
    line-height: 1.6;
    -webkit-font-smoothing: antialiased;
    padding: 0 24px 80px;
    min-height: 100vh;
  }
  .wrap { max-width: 1120px; margin: 0 auto; }

  /* ---- Header Card ---- */
  .header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 20px 24px;
    margin: 28px 0 24px;
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: 0 4px 20px -2px rgba(15, 23, 42, 0.05);
  }
  .brand-group { display: flex; align-items: center; gap: 14px; }
  .logo {
    width: 38px; height: 38px; border-radius: 10px;
    background: linear-gradient(135deg, #4f46e5 0%, #6366f1 100%);
    box-shadow: 0 4px 12px rgba(79, 70, 229, 0.25);
    display: grid; place-items: center;
    font-family: var(--mono); font-weight: 800; font-size: 13px;
    color: #fff; letter-spacing: 0.5px; flex: none;
  }
  .header-titles h1 {
    font-size: 18px; font-weight: 750; letter-spacing: -0.3px;
    color: var(--text); line-height: 1.2;
  }
  .header-titles .crumb {
    display: flex; align-items: center; gap: 6px;
    font-size: 12.5px; color: var(--muted); margin-top: 3px;
  }
  .header-titles .crumb b { color: var(--primary); font-weight: 600; font-family: var(--mono); }
  .header-badges { display: flex; align-items: center; gap: 8px; }
  .badge-chip {
    display: inline-flex; align-items: center; gap: 5px;
    font-family: var(--mono); font-size: 11px; font-weight: 600;
    padding: 4px 10px; border-radius: var(--radius-pill);
    background: var(--surface-subtle); border: 1px solid var(--border);
    color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px;
  }
  .badge-status {
    background: var(--success-soft);
    border-color: var(--success-border);
    color: #065f46;
  }
  .badge-status .status-dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: var(--success);
    box-shadow: 0 0 6px rgba(5, 150, 105, 0.4);
  }

  /* ---- Section Headers ---- */
  .section-header {
    display: flex; align-items: baseline; justify-content: space-between;
    margin: 32px 0 12px;
  }
  h2 {
    font-size: 12px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 1.4px; color: var(--muted);
    display: flex; align-items: center; gap: 8px;
  }
  h2::before {
    content: ""; display: inline-block; width: 4px; height: 12px;
    background: var(--primary); border-radius: 2px;
  }

  /* ---- KPI Metric Cards ---- */
  .metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 14px;
    margin-bottom: 14px;
  }
  .metric-card {
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    padding: 18px 20px;
    position: relative;
    overflow: hidden;
    transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
    box-shadow: 0 2px 10px rgba(15, 23, 42, 0.04);
  }
  .metric-card:hover {
    transform: translateY(-2px);
    border-color: var(--border-strong);
    box-shadow: 0 8px 24px -4px rgba(15, 23, 42, 0.08);
  }
  .metric-card::before {
    content: ""; position: absolute; top: 0; left: 0; right: 0; height: 3px;
    background: var(--card-accent, var(--primary));
  }
  .metric-events { --card-accent: var(--primary); }
  .metric-errors { --card-accent: var(--danger); }
  .metric-warnings { --card-accent: var(--warning); }
  .metric-fatal { --card-accent: #ec4899; }

  .metric-top { display: flex; align-items: center; justify-content: space-between; }
  .metric-card .lbl {
    font-size: 11px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 1px; color: var(--muted);
  }
  .metric-card .num {
    font-size: 32px; font-weight: 800; letter-spacing: -0.8px;
    margin-top: 8px; font-variant-numeric: tabular-nums; line-height: 1.1;
    color: var(--text);
  }
  .metric-errors .num { color: var(--danger); }
  .metric-warnings .num { color: var(--warning); }
  .metric-fatal .num { color: #be185d; }
  .metric-sub { font-size: 11.5px; color: var(--faint); margin-top: 6px; }

  /* ---- Meta Details Strip ---- */
  .meta-strip {
    display: flex; flex-wrap: wrap; align-items: center; gap: 16px 28px;
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    padding: 14px 20px;
    font-size: 12.5px;
    box-shadow: 0 2px 10px rgba(15, 23, 42, 0.03);
  }
  .meta-item { display: flex; align-items: center; gap: 8px; }
  .meta-item dt { color: var(--faint); font-weight: 600; text-transform: uppercase; font-size: 10.5px; letter-spacing: 0.8px; }
  .meta-item dd { color: var(--text); font-weight: 500; }
  .meta-item code {
    font-family: var(--mono); font-size: 11.5px;
    background: var(--surface-subtle); padding: 2px 7px;
    border-radius: 4px; border: 1px solid var(--border);
    color: var(--muted);
  }

  /* ---- Panels Base ---- */
  .panel {
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 22px 24px;
    box-shadow: 0 4px 20px -2px rgba(15, 23, 42, 0.04);
  }

  /* ---- Diagnosis Hero Card ---- */
  .dx-hero {
    background: linear-gradient(145deg, #ffffff 0%, #f8fafc 100%);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 26px 28px;
    position: relative;
    overflow: hidden;
    box-shadow: 0 4px 24px -2px rgba(15, 23, 42, 0.06);
  }
  .dx-hero::before {
    content: ""; position: absolute; top: 0; left: 0; width: 5px; height: 100%;
    background: var(--sev-color, var(--primary));
  }
  .sev-critical { --sev-color: var(--danger); border-color: var(--danger-border); box-shadow: 0 8px 30px -4px rgba(225, 29, 72, 0.08); }
  .sev-high { --sev-color: var(--warning); border-color: var(--warning-border); box-shadow: 0 8px 30px -4px rgba(217, 119, 6, 0.08); }
  .sev-medium { --sev-color: #eab308; border-color: #fef08a; }
  .sev-low { --sev-color: var(--success); border-color: var(--success-border); }

  .dx-head {
    display: flex; align-items: center; justify-content: space-between;
    flex-wrap: wrap; gap: 10px; margin-bottom: 8px;
  }
  .dx-label {
    font-size: 11px; font-weight: 750; text-transform: uppercase;
    letter-spacing: 1.4px; color: var(--faint);
  }
  .dx-hero h3 {
    font-size: 24px; font-weight: 800; letter-spacing: -0.4px;
    color: var(--text); margin: 4px 0 12px;
  }
  .pill {
    display: inline-flex; align-items: center; gap: 6px;
    font-size: 11px; font-weight: 750; text-transform: uppercase;
    letter-spacing: 0.8px; padding: 4px 12px; border-radius: var(--radius-pill);
    border: 1px solid transparent;
  }
  .pill-critical {
    background: var(--danger-soft); color: #be123c;
    border-color: var(--danger-border);
  }
  .pill-high, .pill-medium {
    background: var(--warning-soft); color: #92400e;
    border-color: var(--warning-border);
  }
  .pill-low {
    background: var(--success-soft); color: #065f46;
    border-color: var(--success-border);
  }
  .pill-insufficient {
    background: var(--warning-soft); color: #92400e;
    border-color: var(--warning-border);
  }

  /* Confidence Meter */
  .conf-container {
    margin: 18px 0 24px;
    background: #f8fafc;
    padding: 14px 18px; border-radius: var(--radius-md);
    border: 1px solid var(--border);
  }
  .conf-row {
    display: flex; justify-content: space-between; align-items: baseline;
    font-size: 12.5px; margin-bottom: 8px;
  }
  .conf-label { color: var(--muted); font-weight: 600; }
  .conf-num {
    color: var(--text); font-weight: 750; font-family: var(--mono);
    font-size: 14px; font-variant-numeric: tabular-nums;
  }
  .conf-track {
    height: 8px; background: #e2e8f0;
    border-radius: var(--radius-pill); overflow: hidden;
  }
  .conf-fill {
    height: 100%; border-radius: var(--radius-pill);
    background: linear-gradient(90deg, #4f46e5, #e11d48);
    transition: width 0.4s ease;
  }
  .sev-high .conf-fill { background: linear-gradient(90deg, #4f46e5, #d97706); }
  .sev-low .conf-fill { background: linear-gradient(90deg, #4f46e5, #059669); }

  /* Sub-sections within Diagnosis */
  .dx-section-title {
    font-size: 12px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 1px; color: var(--muted);
    margin: 22px 0 12px; display: flex; align-items: center; gap: 8px;
  }
  .ev-grid { display: grid; gap: 8px; list-style: none; }
  .ev-card {
    display: flex; align-items: center; gap: 14px;
    padding: 12px 16px; border-radius: var(--radius-md);
    background: #ffffff;
    border: 1px solid var(--border);
    font-size: 13px;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }
  .ev-card:hover {
    border-color: var(--border-strong);
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  }
  .ev-card time {
    font-family: var(--mono); font-size: 11.5px; color: var(--faint);
    background: var(--surface-subtle); padding: 3px 8px;
    border-radius: 4px; flex: none; font-variant-numeric: tabular-nums;
  }
  .badge {
    font-size: 10px; font-weight: 750; letter-spacing: 0.6px;
    padding: 3px 9px; border-radius: 6px; flex: none;
    text-transform: uppercase; font-family: var(--mono);
    border: 1px solid transparent;
  }
  .b-error { background: var(--danger-soft); color: #be123c; border-color: var(--danger-border); }
  .b-anomaly { background: var(--warning-soft); color: #92400e; border-color: var(--warning-border); }
  .b-downstream { background: var(--primary-soft); color: #3730a3; border-color: var(--primary-border); }
  .b-temporal { background: var(--info-soft); color: #0369a1; border-color: var(--info-border); }
  .b-stack { background: var(--surface-subtle); color: var(--muted); border-color: var(--border); }
  .ev-card p { color: var(--text); margin: 0; font-weight: 500; }

  /* Recommendations Checklist */
  .recs-list { list-style: none; display: grid; gap: 8px; }
  .rec-card {
    display: flex; gap: 14px; align-items: center;
    padding: 12px 16px; border-radius: var(--radius-md);
    background: #ffffff;
    border: 1px solid var(--border);
    font-size: 13.5px;
  }
  .rec-num {
    width: 24px; height: 24px; border-radius: 50%;
    background: var(--primary-soft); color: var(--primary);
    border: 1px solid var(--primary-border);
    display: grid; place-items: center;
    font-family: var(--mono); font-size: 11.5px; font-weight: 750;
    flex: none;
  }
  .rec-card span:last-child { color: var(--text); font-weight: 500; }

  /* ---- Timeline Panel ---- */
  .timeline-head {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 16px;
  }
  .legend {
    display: flex; gap: 16px; font-size: 12px; color: var(--muted);
  }
  .legend-item { display: flex; align-items: center; gap: 6px; }
  .legend-dot { width: 8px; height: 8px; border-radius: 50%; }
  .legend-normal { background: var(--primary); }
  .legend-anomaly { background: var(--danger); }

  /* ---- Anomalies Cards ---- */
  .anom-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 12px;
  }
  .acard {
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    padding: 16px 18px;
    box-shadow: 0 2px 10px rgba(15, 23, 42, 0.03);
    transition: transform 0.15s ease, border-color 0.15s ease;
  }
  .acard:hover {
    transform: translateY(-2px);
    border-color: var(--warning);
    box-shadow: 0 6px 18px -2px rgba(217, 119, 6, 0.1);
  }
  .acard-head {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 10px;
  }
  .acard .atime {
    font-family: var(--mono); font-size: 12px; font-weight: 700;
    color: var(--muted);
  }
  .acard .surge-pill {
    font-family: var(--mono); font-size: 11px; font-weight: 800;
    background: var(--warning-soft); color: #92400e;
    padding: 2px 8px; border-radius: var(--radius-pill);
    border: 1px solid var(--warning-border);
  }
  .acard .stat-row {
    display: flex; justify-content: space-between; font-size: 12px;
    color: var(--muted); margin-top: 5px;
  }
  .acard .stat-row b {
    color: var(--text); font-variant-numeric: tabular-nums;
    font-family: var(--mono);
  }

  /* ---- Error Groups Cards ---- */
  .eg-list { display: grid; gap: 10px; }
  .gcard {
    background: var(--surface-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    padding: 16px 20px;
    transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
    box-shadow: 0 2px 10px rgba(15, 23, 42, 0.03);
  }
  .gcard:hover {
    border-color: var(--border-strong);
    box-shadow: 0 8px 24px -4px rgba(15, 23, 42, 0.08);
    transform: translateY(-1px);
  }
  .gcard-header {
    display: flex; align-items: center; justify-content: space-between;
    gap: 12px; margin-bottom: 10px;
  }
  .gcard-rank {
    display: inline-flex; align-items: center; gap: 6px;
    font-family: var(--mono); font-size: 11.5px; font-weight: 750;
    color: var(--faint);
  }
  .gcard-count {
    font-family: var(--mono); font-size: 12px; font-weight: 750;
    background: var(--danger-soft); color: #be123c;
    border: 1px solid var(--danger-border);
    padding: 3px 10px; border-radius: var(--radius-pill);
    font-variant-numeric: tabular-nums;
  }
  .gcard-time {
    font-family: var(--mono); font-size: 11.5px; color: var(--faint);
    margin-left: auto;
  }
  .gcard-msg {
    font-family: var(--mono); font-size: 13px; color: var(--text);
    background: var(--surface-code);
    border: 1px solid var(--border);
    padding: 10px 14px; border-radius: var(--radius-sm);
    word-break: break-all; line-height: 1.5; font-weight: 600;
  }
  .gcard-example {
    margin-top: 10px; font-family: var(--mono); font-size: 11.5px;
    color: var(--muted); background: var(--surface-subtle);
    padding: 8px 12px; border-radius: 6px;
    border-left: 3px solid var(--primary);
    overflow-x: auto;
  }

  /* ---- Source Files & Warnings ---- */
  .files-list { list-style: none; display: grid; gap: 6px; }
  .files-list li {
    background: var(--surface-code); border: 1px solid var(--border);
    padding: 8px 14px; border-radius: var(--radius-sm);
  }
  .files-list code { font-family: var(--mono); font-size: 12px; color: var(--muted); }
  .warn-banner {
    background: var(--warning-soft);
    border: 1px solid var(--warning-border);
    border-radius: var(--radius-md);
    padding: 16px 20px;
  }
  .warn-banner ul { margin: 0; padding-left: 20px; color: #92400e; font-size: 13px; }

  /* ---- Footer ---- */
  footer {
    text-align: center; margin-top: 50px; padding-top: 20px;
    border-top: 1px solid var(--border);
    font-size: 12px; color: var(--faint);
  }

  @media (max-width: 640px) {
    body { padding: 0 16px 40px; }
    .header { flex-direction: column; align-items: flex-start; gap: 14px; }
    .header-badges { align-self: flex-start; }
    .meta-strip { flex-direction: column; align-items: flex-start; gap: 8px; }
  }

  @media print {
    body { background: #fff !important; color: #000 !important; }
    .metric-card, .dx-hero, .panel, .acard, .gcard {
      background: #fff !important; color: #000 !important;
      border: 1px solid #ccc !important; box-shadow: none !important;
    }
  }
</style>
</head>
<body>
<div class="wrap">
  <!-- Header Bar -->
  <header class="header">
    <div class="brand-group">
      <div class="logo">FL</div>
      <div class="header-titles">
        <h1>FaultLens Report</h1>
        <div class="crumb">
          <span>Incident Diagnosis</span>
          <span style="color:var(--faint)">/</span>
          <b>{{.Summary.Source}}</b>
        </div>
      </div>
    </div>
    <div class="header-badges">
      <div class="badge-chip"><span style="color:var(--primary)">fmt:</span> {{.Summary.Format}}</div>
      <div class="badge-chip badge-status">
        <span class="status-dot"></span>
        <span>Analyzed</span>
      </div>
    </div>
  </header>

  <!-- Summary KPI Cards -->
  <div class="metrics">
    <div class="metric-card metric-events">
      <div class="metric-top">
        <div class="lbl">Events</div>
      </div>
      <div class="num">{{.Summary.Events}}</div>
      <div class="metric-sub">Total log entries processed</div>
    </div>
    <div class="metric-card metric-errors">
      <div class="metric-top">
        <div class="lbl">Errors</div>
      </div>
      <div class="num">{{.Summary.Errors}}</div>
      <div class="metric-sub">Error events encountered</div>
    </div>
    <div class="metric-card metric-warnings">
      <div class="metric-top">
        <div class="lbl">Warnings</div>
      </div>
      <div class="num">{{.Summary.Warnings}}</div>
      <div class="metric-sub">Potential warning signals</div>
    </div>
    <div class="metric-card metric-fatal">
      <div class="metric-top">
        <div class="lbl">Fatal</div>
      </div>
      <div class="num">{{.Summary.Fatal}}</div>
      <div class="metric-sub">Critical crashes / panics</div>
    </div>
  </div>

  <!-- Meta Details Strip -->
  <dl class="meta-strip">
    {{if not .Summary.FirstEvent.IsZero}}
    <div class="meta-item">
      <dt>Window</dt>
      <dd>{{.Summary.FirstEvent.Format "2006-01-02 15:04:05"}} → {{.Summary.LastEvent.Format "2006-01-02 15:04:05"}}</dd>
    </div>
    {{end}}
    <div class="meta-item">
      <dt>Format</dt>
      <dd>{{.Summary.Format}}</dd>
    </div>
    <div class="meta-item">
      <dt>Source</dt>
      <dd><code>{{.Summary.Source}}</code></dd>
    </div>
    {{if gt .Summary.ParsingWarnings 0}}
    <div class="meta-item">
      <dt>Warnings</dt>
      <dd style="color:var(--warning)">{{.Summary.ParsingWarnings}} lines could not be parsed</dd>
    </div>
    {{end}}
  </dl>

  <!-- Diagnosis Hero Card -->
  <div class="section-header">
    <h2>Diagnosis</h2>
  </div>
  <section class="dx-hero sev-{{.Diagnosis.Severity}}">
    <div class="dx-head">
      <span class="dx-label">Root Cause Hypothesis</span>
      <span class="pill pill-{{.Diagnosis.Severity}}">{{.Diagnosis.Severity}}</span>
    </div>
    <h3>{{.Diagnosis.RootCause}}</h3>

    {{if ne .Diagnosis.RootCause "Insufficient evidence"}}
    <div class="conf-container">
      <div class="conf-row">
        <span class="conf-label">Confidence Score</span>
        <span class="conf-num">{{.ConfidencePct}}%</span>
      </div>
      <div class="conf-track">
        <div class="conf-fill" style="width:{{.ConfidencePct}}%"></div>
      </div>
    </div>
    {{end}}

    {{if .Diagnosis.Evidence}}
    <div class="dx-section-title">Evidence</div>
    <ul class="ev-grid">
      {{range .Diagnosis.Evidence}}
      <li class="ev-card">
        {{if not .Timestamp.IsZero}}<time>{{.Timestamp.Format "15:04:05"}}</time>{{end}}
        <span class="badge b-{{badgeClass .Type}}">{{.Type}}</span>
        <p>{{.Message}}</p>
      </li>
      {{end}}
    </ul>
    {{end}}

    {{if .Diagnosis.Recommendations}}
    <div class="dx-section-title">Recommended Actions</div>
    <ol class="recs-list">
      {{range $i, $r := .Diagnosis.Recommendations}}
      <li class="rec-card">
        <span class="rec-num">{{add $i 1}}</span>
        <span>{{$r}}</span>
      </li>
      {{end}}
    </ol>
    {{end}}
  </section>

  <!-- Error Timeline Panel -->
  {{if .TimelineSVG}}
  <div class="section-header">
    <h2>Timeline</h2>
  </div>
  <div class="panel">
    <div class="timeline-head">
      <span style="font-size:12.5px;color:var(--muted)">Error volume over time (per-minute buckets)</span>
      <div class="legend">
        <div class="legend-item"><span class="legend-dot legend-normal"></span><span>Normal</span></div>
        <div class="legend-item"><span class="legend-dot legend-anomaly"></span><span>Anomaly spike</span></div>
      </div>
    </div>
    {{.TimelineSVG}}
  </div>
  {{end}}

  <!-- Anomalies Section -->
  {{if .Anomalies}}
  <div class="section-header">
    <h2>Anomalies</h2>
  </div>
  <div class="anom-grid">
    {{range .Anomalies}}
    <div class="acard">
      <div class="acard-head">
        <span class="atime">{{.Bucket.Format "2006-01-02 15:04"}}</span>
        <span class="surge-pill">{{printf "%.1f" .Increase}}x</span>
      </div>
      <div class="stat-row"><span>Baseline Mean</span><b>{{printf "%.1f" .BaselineMean}}</b></div>
      <div class="stat-row"><span>Errors in Bucket</span><b>{{.Current}}</b></div>
      <div class="stat-row"><span>Z-Score</span><b>{{if isInf .ZScore}}∞{{else}}{{printf "%.1f" .ZScore}}{{end}}</b></div>
    </div>
    {{end}}
  </div>
  {{end}}

  <!-- Error Groups Section -->
  <div class="section-header">
    <h2>Error Groups</h2>
  </div>
  {{if .ErrorGroups}}
  <div class="eg-list">
    {{range $i, $g := .ErrorGroups}}
    <div class="gcard">
      <div class="gcard-header">
        <span class="gcard-rank">#{{add $i 1}}</span>
        <span class="gcard-count">{{$g.Count}} occurrences</span>
        {{if not $g.FirstSeen.IsZero}}
        <span class="gcard-time">First: {{$g.FirstSeen.Format "15:04:05"}} · Last: {{$g.LastSeen.Format "15:04:05"}}</span>
        {{end}}
      </div>
      <div class="gcard-msg">{{$g.Message}}</div>
      {{if $g.Examples}}
      <div class="gcard-example">{{(index $g.Examples 0).Raw}}</div>
      {{end}}
    </div>
    {{end}}
  </div>
  {{else}}
  <div class="panel" style="text-align:center;color:var(--faint);font-size:13px;padding:32px;">
    No error groups detected
  </div>
  {{end}}

  <!-- Source Files Section -->
  {{if .SourceFiles}}
  <div class="section-header">
    <h2>Source Files</h2>
  </div>
  <div class="panel">
    <ul class="files-list">
      {{range .SourceFiles}}
      <li><code>{{.}}</code></li>
      {{end}}
    </ul>
  </div>
  {{end}}

  <!-- Configuration Warnings Section -->
  {{if .ConfigWarnings}}
  <div class="section-header">
    <h2>Configuration Warnings</h2>
  </div>
  <div class="warn-banner">
    <ul>
      {{range .ConfigWarnings}}
      <li>{{.}}</li>
      {{end}}
    </ul>
  </div>
  {{end}}

  <!-- Footer -->
  <footer>
    FaultLens • Local-first, explainable log incident diagnosis • Offline report
  </footer>
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
		"isInf":      func(f float64) bool { return math.IsInf(f, 0) },
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

// timelineSVG renders the per-minute error counts as a sleek area chart (fill +
// line), with anomalous buckets marked with a glowing target marker — a
// modern Datadog/Grafana-style metric view.
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
		h := float64(bucket.Errors) / float64(maxErr) * (plotH - 14)
		pts[i] = pt{x: x, y: plotH - h}
	}
	base := float64(plotH)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg viewBox="0 0 %d %d" role="img" aria-label="Error timeline" style="width:100%%;height:auto;display:block">`, width, height)

	// SVG Gradients for light mode
	b.WriteString(`<defs>`)
	b.WriteString(`<linearGradient id="areaGrad" x1="0" y1="0" x2="0" y2="1">`)
	b.WriteString(`<stop offset="0%" stop-color="#4f46e5" stop-opacity="0.18"/>`)
	b.WriteString(`<stop offset="100%" stop-color="#4f46e5" stop-opacity="0.01"/>`)
	b.WriteString(`</linearGradient>`)
	b.WriteString(`</defs>`)

	// Horizontal grid lines with y-axis labels (max / half / zero).
	for _, frac := range []float64{1, 0.5, 0} {
		y := plotH - frac*(plotH-14)
		op := "1"
		dash := ` stroke-dasharray="3,3"`
		strokeColor := "#e2e8f0"
		if frac == 0 {
			strokeColor = "#cbd5e1"
			dash = ""
		}
		fmt.Fprintf(&b, `<line x1="0" y1="%.1f" x2="%d" y2="%.1f" stroke="%s" stroke-opacity="%s"%s/>`, y, width, y, strokeColor, op, dash)
		val := int(float64(maxErr)*frac + 0.5)
		fmt.Fprintf(&b, `<text x="4" y="%.1f" font-size="9.5" fill="#64748b" font-family="ui-monospace,SFMono-Regular,Consolas,monospace">%d</text>`, y-4, val)
	}

	// Area fill under the line (smooth gradient)
	var area strings.Builder
	area.WriteString("M")
	for i, p := range pts {
		fmt.Fprintf(&area, " %.1f,%.1f", p.x, p.y)
		if i < n-1 {
			area.WriteString(" L")
		}
	}
	fmt.Fprintf(&area, " L%.1f,%.1f L%.1f,%.1f Z", pts[n-1].x, base, pts[0].x, base)
	fmt.Fprintf(&b, `<path d="%s" fill="url(#areaGrad)"/>`, area.String())

	// Line connecting the points.
	var line strings.Builder
	line.WriteString("M")
	for i, p := range pts {
		fmt.Fprintf(&line, " %.1f,%.1f", p.x, p.y)
		if i < n-1 {
			line.WriteString(" L")
		}
	}
	fmt.Fprintf(&b, `<path d="%s" fill="none" stroke="#4f46e5" stroke-width="2.5" stroke-linejoin="round" stroke-linecap="round"/>`, line.String())

	// Normal points
	for _, p := range pts {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="2" fill="#4f46e5"/>`, p.x, p.y)
	}

	// Anomalous buckets: double concentric glow markers.
	for i, p := range pts {
		if anomAt[buckets[i].Start.Unix()] {
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="8" fill="#e11d48" fill-opacity="0.2"/>`, p.x, p.y)
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="4" fill="#e11d48" stroke="#ffffff" stroke-width="1.5"/>`, p.x, p.y)
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
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" font-size="10.5" fill="#64748b" text-anchor="%s" font-family="ui-monospace,SFMono-Regular,Consolas,monospace">%s</text>`,
			pts[idx].x, height-4, anchor, buckets[idx].Start.Format("15:04"))
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

