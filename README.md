# FaultLens

> See beyond the error.

**🌐 Language:** [English](README.md) | [简体中文](README.zh-CN.md)

FaultLens is a **local-first, offline-first** log incident diagnosis CLI. It
identifies error patterns, detects anomalous time points, analyzes the
relationships between errors, and infers the most likely root cause from
transparent, explainable rules — no network, no database, no AI API required.

> **Status: WIP** — V1 is under active development. Command behavior and output
> formats below are the target design and may change until the V1 release.

## Features

- Detect error patterns in plain text, JSON, Java/Spring Boot, and Nginx logs
- Normalize dynamic values (IPs, ports, UUIDs, numbers, timestamps, URLs…)
- Cluster errors by stable fingerprints
- Build per-minute timelines and detect anomalies with explainable baselines
- Diagnose the most likely root cause with evidence and confidence
- Output reports as terminal text, JSON, or Markdown

*Full feature list is being finalized in V1.*

## Installation

> TODO: documented once a stable release is available.

```bash
go install github.com/faultlens/faultlens/cmd/faultlens@latest
```

## Quick Start

```bash
faultlens app.log
```

Or pipe logs from stdin:

```bash
cat app.log | faultlens
```

## Supported Logs

- Plain text (timestamp + level + message)
- JSON (JSONL)
- Java / Spring Boot (including stack traces)
- Nginx access and error logs

## Example

> Target output (V1 in development):

```text
FaultLens
────────────────────────────────────────────

Log Summary

Events:       182,391
Errors:       4,381
Warnings:     8,212
Time Range:   14:00 - 15:00
Format:       Java

────────────────────────────────────────────

Anomalies

14:32
Error rate increased 41.2x

────────────────────────────────────────────

Diagnosis

ROOT CAUSE
MySQL became unavailable

Confidence
0.91

Evidence
14:32:15  ERROR_PATTERN  MySQL connection refused
14:32:19  ANOMALY        HTTP 500 increased 42x

Recommendations
1. Check MySQL availability
2. Check database connection limit
```

## How It Works

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

Every diagnosis is backed by evidence, a rule, and a confidence score that can
be traced back to the log lines that produced it. If the evidence is not
strong enough, FaultLens reports `Insufficient evidence` instead of guessing.

## Architecture

```
cmd/faultlens/   CLI entry point (Cobra)
internal/model/  shared data model
internal/input/  file & stdin input
internal/parser/ log format parsers
internal/normalize/ error normalization
internal/grouping/  error clustering
internal/timeline/  time bucket analysis
internal/anomaly/   anomaly detection
internal/diagnosis/ root cause engine + rules
internal/output/    terminal / JSON / Markdown renderers
```

Core analysis packages are independent of the CLI so they can be reused by
future GitHub Actions or API integrations.

## CLI Reference

> Commands below are the V1 target. Only `version` is implemented so far.

- `faultlens <file>` — full analysis report (default)
- `faultlens errors <file>` — error clustering
- `faultlens timeline <file>` — timeline analysis
- `faultlens incident <file>` — incident diagnosis
- `faultlens version` — print version

## Output Formats

- `terminal` — human-readable report
- `json` — stable schema for tooling and CI
- `markdown` — for issues, postmortems, and PR comments

## Development

```bash
go build ./...
go test ./...
go vet ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Testing

> TODO: describe the test layout and fixtures once populated.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE)