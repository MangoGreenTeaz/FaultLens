# FaultLens

> See beyond the error.

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

> TODO: add a full annotated example report.

## How It Works

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

Every diagnosis is backed by evidence, a rule, and a confidence score that can
be traced back to the log lines that produced it.

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

## CLI Reference

> TODO: document all commands and flags once implemented.

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