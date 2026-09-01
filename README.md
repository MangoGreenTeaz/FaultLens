# FaultLens

> See beyond the error.

**🌐 Language:** [English](README.md) | [简体中文](README.zh-CN.md)

FaultLens is a **local-first, offline-first** log incident diagnosis CLI. It
identifies error patterns, detects anomalous time points, analyzes the
relationships between errors, and infers the most likely root cause from
transparent, explainable rules — no network, no database, no AI API required.

## Features

- Detect error patterns in plain text, JSON, Java/Spring Boot, and Nginx logs
- Normalize dynamic values (IPs, ports, UUIDs, numbers, timestamps, URLs…)
- Cluster errors by stable SHA-256 fingerprints
- Build per-minute timelines and detect anomalies with explainable baselines
- Diagnose the most likely root cause with evidence and confidence
- Report `Insufficient evidence` instead of guessing when signals are weak
- Output reports as terminal text, JSON, or Markdown

## Installation

Requires [Go 1.22+](https://go.dev/dl/).

```bash
go install github.com/faultlens/faultlens/cmd/faultlens@latest
```

Or build from source:

```bash
git clone https://github.com/faultlens/faultlens.git
cd faultlens
go build ./cmd/faultlens
```

## Quick Start

```bash
faultlens app.log
```

Or pipe logs from stdin:

```bash
cat app.log | faultlens
docker logs some-container | faultlens incident
```

## Supported Logs

- Plain text (`timestamp + level + message`)
- JSON (JSONL), with common field aliases (`timestamp`/`time`/`ts`, `level`/`severity`, …)
- Java / Spring Boot, including multi-line stack traces aggregated into a single event
- Nginx access and error logs

The format is auto-detected from content; use `--format` to force it.

## Example

```bash
faultlens incident testdata/incidents/mysql-outage.log
```

```text
Incident detected

Root Cause:
Database unavailable

Confidence:
90%

Severity:
critical

Evidence:
14:32:02  database-related errors detected
14:32:01  connection failures detected
14:32:05  HTTP 5xx errors observed downstream
14:32:05  database errors preceded HTTP 5xx spike

Recommended:
1. Check MySQL availability
2. Check database connection limit
3. Check recent database restart
4. Check network connectivity between application and database
```

## How It Works

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

Every diagnosis is backed by evidence, a rule, and a confidence score that can
be traced back to the log lines that produced it. Confidence is assembled from
explainable building blocks (strong evidence, supporting evidence, temporal
correlation, downstream impact, contradictions) and clamped to `[0, 1]`.

## Architecture

```
cmd/faultlens/       CLI entry point (Cobra)
internal/engine/     analysis pipeline (input → … → diagnosis)
internal/model/      shared data model
internal/input/      file & stdin streaming input
internal/parser/     plain / json / java / nginx / auto parsers
internal/normalize/  error normalization
internal/grouping/   error clustering + fingerprints
internal/timeline/   per-minute time buckets
internal/anomaly/    z-score anomaly detection
internal/diagnosis/  root-cause engine + 6 rules
internal/output/     terminal / JSON / Markdown renderers
testdata/            sample logs and incident fixtures
```

Core analysis packages are independent of the CLI so they can be reused by
future GitHub Actions or API integrations.

## CLI Reference

```bash
faultlens <file>          # full analysis report (default)
faultlens errors <file>   # error clustering
faultlens timeline <file> # per-minute timeline + anomalies
faultlens incident <file> # root-cause diagnosis
faultlens version         # print version
```

Global flags:

```bash
--format auto|plain|json|java|nginx   # log format (default: auto)
--output terminal|json|markdown       # report format (default: terminal)
--from <RFC3339>                      # only events at/after this time
--to   <RFC3339>                      # only events at/before this time
```

## Output Formats

- `terminal` — human-readable report
- `json` — stable schema (`summary`, `error_groups`, `timeline`, `anomalies`, `diagnosis`) for tooling and CI
- `markdown` — for issues, postmortems, and PR comments

## Development

```bash
go build ./...
go test ./...
go vet ./...
```

## Testing

```bash
go test ./...          # unit tests for every core package
go test ./internal/engine/ -run TestIncidentFixtures -v   # end-to-end runs
```

Every core package has table-driven unit tests, and `testdata/incidents/*`
drive end-to-end pipeline tests (e.g. `mysql-outage.log` must diagnose
`Database unavailable`, never the HTTP 5xx symptom).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)