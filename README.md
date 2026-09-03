# FaultLens

> See beyond the error.

**🌐 Language:** [English](README.md) | [简体中文](README.zh-CN.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://go.dev/dl/)
[![CI](https://github.com/MangoGreenTeaz/FaultLens/actions/workflows/test.yml/badge.svg)](https://github.com/MangoGreenTeaz/FaultLens/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MangoGreenTeaz/FaultLens)](https://goreportcard.com/report/github.com/MangoGreenTeaz/FaultLens)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

FaultLens is a **local-first, offline-first** log incident diagnosis CLI. It
identifies error patterns, detects anomalous time points, analyzes the
relationships between errors, and infers the most likely root cause from
transparent, explainable rules — **no network, no database, no AI API
required.**

```bash
faultlens app.log
# or
cat app.log | faultlens
```

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Supported Logs](#supported-logs)
- [Example](#example)
- [How It Works](#how-it-works)
- [Architecture](#architecture)
- [CLI Reference](#cli-reference)
- [Output Formats](#output-formats)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## ✨ Features

- 🔍 **Unified parsing** — plain text, JSON (with field aliases), Java/Spring
  Boot (multi-line stack traces merged into one event), and Nginx access /
  error logs. Format is auto-detected from content.
- 🧹 **Dynamic value normalization** — IPs, ports, UUIDs, numbers, timestamps,
  URLs, paths and hex IDs become stable placeholders, while diagnostic values
  such as HTTP status codes are preserved.
- 🗂️ **Error clustering** — semantically identical errors merge into groups
  keyed by stable SHA-256 fingerprints, ranked by occurrence count.
- 📈 **Anomaly detection** — per-minute time buckets compared against an
  explainable baseline (mean + standard deviation, z-score). No machine
  learning, fully transparent.
- 🧠 **Explainable diagnosis** — every root cause ships with evidence, a rule,
  and a confidence score traceable back to the log lines. When evidence is
  weak, FaultLens reports `Insufficient evidence` instead of guessing.
- 📦 **Multiple report formats** — terminal text, stable JSON (for CI and
  tooling), and Markdown (for issues, postmortems and PR comments).
- 🔒 **Local-first & offline** — everything runs on your machine. No telemetry,
  no external calls.

## 📦 Installation

**Requirements:** [Go 1.22+](https://go.dev/dl/)

Install the latest release:

```bash
go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest
```

Or build from source:

```bash
git clone https://github.com/MangoGreenTeaz/FaultLens.git
cd FaultLens
go build -o faultlens ./cmd/faultlens
```

Verify the install:

```bash
faultlens version
```

## 🚀 Quick Start

Analyze a log file:

```bash
faultlens app.log
```

Analyze specific aspects:

```bash
faultlens errors app.log     # error clustering
faultlens timeline app.log   # per-minute timeline + anomalies
faultlens incident app.log   # root-cause diagnosis
```

Pipe logs from stdin:

```bash
cat app.log | faultlens
docker logs some-container | faultlens incident
```

Filter by time window:

```bash
faultlens incident app.log --from 2026-08-31T14:00:00Z --to 2026-08-31T15:00:00Z
```

## 📊 Supported Logs

| Format | Example | Notes |
| --- | --- | --- |
| Plain text | `2026-08-31 14:32:01 ERROR database connection failed` | `timestamp + level + message` |
| JSON | `{"timestamp":"2026-08-31T14:32:01Z","level":"ERROR","message":"..."}` | JSONL; aliases: `timestamp/time/ts`, `level/severity`, `message/msg`, `service/service_name` |
| Java / Spring Boot | `2026-08-31T14:32:01.123+08:00 ERROR ...` + stack trace | Stack traces (incl. `Caused by:`, `Suppressed:`) aggregate into a single event |
| Nginx | `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 500 123` | Access + error logs; status/method/path extracted into fields |

Use `--format auto|plain|json|java|nginx` to override content-based detection.

## 📝 Example

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

## ⚙️ How It Works

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

1. **Parse** raw log lines into a unified `LogEvent` model.
2. **Normalize** dynamic values so `Connection refused 10.0.0.1:3306` and
   `Connection refused 10.0.0.2:3306` become the same shape.
3. **Group** identical errors by fingerprint and count them.
4. **Build** per-minute buckets and **detect** anomalous spikes against an
   explainable baseline.
5. **Diagnose** by scoring each candidate root cause with evidence and
   confidence; symptoms (HTTP 5xx) never beat their upstream cause.
6. **Report** in the format you choose.

## 🏗️ Architecture

```
cmd/faultlens/       CLI entry point (Cobra)
internal/engine/     analysis pipeline (input → … → diagnosis)
internal/model/      shared data model (LogEvent, Diagnosis, …)
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

Core analysis packages are independent of the CLI, so the pipeline can be
reused by future GitHub Actions or API integrations.

## 💻 CLI Reference

```bash
faultlens <file>          # full analysis report (default)
faultlens errors <file>   # error clustering
faultlens timeline <file> # per-minute timeline + anomalies
faultlens incident <file> # root-cause diagnosis
faultlens diff a.json b.json # compare two JSON reports
faultlens version         # print version
```

| Flag | Values | Default | Description |
| --- | --- | --- | --- |
| `--format` | `auto`, `plain`, `json`, `java`, `nginx` | `auto` | Log format |
| `--output` | `terminal`, `json`, `markdown` | `terminal` | Report format |
| `--from` | RFC 3339 timestamp | — | Only events at/after this time |
| `--to` | RFC 3339 timestamp | — | Only events at/before this time |

## 🖨️ Output Formats

- **terminal** — human-readable report (default).
- **json** — stable schema for tooling and CI:
  `{ "summary", "error_groups", "timeline", "anomalies", "diagnosis" }`.
- **html** — single-file, fully offline report with an inline SVG error
  timeline; suitable for CI artifacts and incident sharing.
- **markdown** — tables and headings, ready for GitHub issues, postmortems and
  PR comments.

## 🤖 GitHub Actions

FaultLens fits naturally into CI/CD pipelines: JSON for machine consumption,
HTML as a shareable artifact.

Analyze logs and produce both artifacts:

```yaml
- name: Analyze logs with FaultLens
  run: |
    faultlens ./logs/app.log --output json > faultlens.json
    faultlens ./logs/app.log --output html -o faultlens-report.html

- name: Upload artifacts
  uses: actions/upload-artifact@v4
  with:
    name: faultlens-analysis
    path: |
      faultlens.json
      faultlens-report.html
```

Or use the official composite action shipped in this repository:

```yaml
- uses: ./.github/actions/faultlens
  with:
    log-file: testdata/incidents/mysql-outage.log
    output: html
    output-file: faultlens-report.html
```

A complete, runnable example is in
[`.github/workflows/ci-analysis.yml`](.github/workflows/ci-analysis.yml) — it
analyzes a fixture, uploads JSON + HTML artifacts, and writes a job summary.

## 🧪 Testing

```bash
go test ./...                                    # unit tests for every package
go test ./internal/engine/ -run TestIncidentFixtures -v   # end-to-end tests
go vet ./...
```

Every core package has table-driven unit tests. The `testdata/incidents/*`
fixtures drive full-pipeline integration tests — for example,
`mysql-outage.log` must diagnose `Database unavailable`, never the HTTP 5xx
symptom.

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md)
first. It covers how to add a parser, add a diagnosis rule, and submit a pull
request.

## 📄 License

FaultLens is released under the [MIT License](LICENSE).

```text
MIT License

Copyright (c) 2026 Hu Lei

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

By contributing to FaultLens, you agree that your contributions will be
licensed under the same MIT License.