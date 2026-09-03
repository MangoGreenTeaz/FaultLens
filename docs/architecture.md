# FaultLens Architecture

This document describes the FaultLens design based on the **current codebase**
(`internal/` packages). Everything here matches the implementation; if you
spot a mismatch, file an issue.

## Overview

FaultLens is a local-first, offline-first log incident diagnosis CLI. It reads
logs from files or stdin, parses them into a unified event model, normalizes
dynamic values, clusters identical errors, builds a timeline, detects
anomalies, and produces an evidence-based root-cause diagnosis.

```
cmd/faultlens/       CLI entry point (Cobra)
internal/engine/     analysis pipeline (input → … → diagnosis)
internal/model/      shared data model
internal/input/      file & stdin streaming input
internal/parser/     format parsers + auto detection
internal/normalize/  dynamic value normalization
internal/grouping/   error clustering + fingerprints
internal/timeline/   per-minute time buckets
internal/anomaly/    z-score anomaly detection
internal/diagnosis/  root-cause engine + rules
internal/output/     terminal / JSON / Markdown / HTML renderers
internal/config/     configuration load, merge and validation
```

The pipeline lives in `internal/engine` and is independent of the CLI, so the
same core can be reused by future GitHub Actions or API integrations.

## Processing Pipeline

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

`internal/engine/analyzer` accumulates events from one or more inputs into a
single `Result`:

1. **Input**: streaming `bufio.Scanner` (tolerates long lines, strips a UTF-8
   BOM, handles CRLF).
2. **Parse**: every line goes through a `parser.Parser`; events are emitted as
   `model.LogEvent`.
3. **Normalize** (inside `grouping`): the event message is rewritten to a
   stable shape.
4. **Group**: identical normalized messages merge into `grouping.ErrorGroup`
   keyed by a SHA-256 fingerprint.
5. **Timeline**: events are folded into per-minute `timeline.Bucket`.
6. **Anomaly**: the bucket series is compared against a baseline (mean +
   standard deviation) and spikes are flagged.
7. **Diagnose**: `diagnosis.Engine` evaluates every rule against the
   pre-computed context and picks the highest-confidence hypothesis.
8. **Report**: an `output` renderer formats the `Result`.

Events are never re-parsed: rules consume pre-computed aggregates
(`ErrorGroups`, `Timeline`, `Anomalies`, and streaming HTTP 5xx statistics).

## Parser Architecture

`internal/parser` defines a stateful interface:

```go
type Parser interface {
    Name() string
    CanParse(line string) bool
    Parse(line string) []*model.LogEvent
    Flush() []*model.LogEvent
    Issues() int
    Reset()
}
```

- `Parse` feeds one line and returns completed events. Continuation lines
  (e.g. Java stack traces) are buffered until a complete event can be emitted.
- `CanParse` reports whether a line *starts* an event in this format.
- `Issues` counts lines the parser could not convert (reported as parsing
  warnings instead of failing).

Built-in parsers:

| Parser | Source file | Highlights |
| --- | --- | --- |
| Plain text | `plain.go` | timestamp + level + message; never fails |
| JSON | `json.go` | field aliases; unknown fields → `Fields` |
| Java / Spring Boot | `java.go` | multi-line stack traces → one event |
| Nginx | `nginx.go` | access (status/method/path) + error logs |
| Apache | `apache.go` | common / combined / vhost-combined |
| Python | `python.go` | standard `logging` (comma-fractional seconds) |
| Syslog | `syslog.go` | RFC 3164 + RFC 5424, PRI → severity |
| Docker JSON | `docker.go` | `{"log","stream","time"}` |
| Kubernetes | `kubernetes.go` | kubelet container logs |
| Auto | `auto.go` | content-based detection, delegates to one of the above |

Auto detection (`detectFormat` in `auto.go`) buffers the first 20 lines and
classifies by content, not extension. Order matters: Docker JSON before JSON,
Kubernetes/Python before Java, etc.

### Adding a Parser

1. Create `internal/parser/<name>.go` implementing `Parser` (stateful `Parse`
   for multi-line events).
2. Register in `detectFormat` (`auto.go`) with the right priority and in
   `newParser` (`internal/engine/engine.go`) for `--format`.
3. Add a fixture under `testdata/<format>/` and table-driven tests covering
   normal, malformed and boundary lines.
4. Add the fixture to `TestFormatFixturesDetected` in
   `internal/engine/integration_test.go`.
5. Update the Supported Log Formats table in the README.

## Normalization

`internal/normalize` rewrites dynamic values in messages into stable
placeholders. Rule order is deliberate — more specific patterns first, the
generic numeric rule last:

```
URL → full timestamp → bare clock time → IPv6 → IP:port → IP → UUID →
Unix path → Windows path → Hex → number
```

- `HTTP 500` and `HTTP 404` are **protected** so status codes keep their
  diagnostic meaning (`_STATUS_<code>_` placeholder, restored afterwards).
- `HTTP/1.1` collapses to `HTTP` so version differences do not split groups.
- IPv6 and wall-clock times are disambiguated by validation.

## Error Fingerprinting

`internal/grouping`:

```go
Fingerprint(normalizedMessage) → SHA-256 hex digest (64 chars)
```

`Grouper.Add` keeps `ERROR` and `FATAL` events, groups them by fingerprint,
tracks count/first-seen/last-seen, and stores at most 3 example events.
Groups are sorted by count descending.

## Timeline / Anomaly Detection

`internal/timeline` folds events into buckets (default 1 minute):

```go
type Bucket struct { Start time.Time; Total, Errors, Warnings, Fatal int }
```

`internal/anomaly` detects spikes without machine learning:

- Baseline = mean and sample standard deviation of **previous** buckets.
- A bucket is flagged when `z-score ≥ 3` and `increase ≥ 3x`, with a minimum
  of 5 baseline buckets and 10 errors (all tunable via config).
- When the baseline has zero variance, the increase multiple alone is used.

## Diagnosis Engine

`internal/diagnosis`:

```go
type DiagnosisRule interface {
    ID() string
    Evaluate(ctx *DiagnosisContext) *model.Diagnosis
}

type DiagnosisContext struct {
    ErrorGroups  []grouping.ErrorGroup
    Timeline     []timeline.Bucket
    Anomalies    []anomaly.Detection
    FiveXXCount  int
    FiveXXFirst  time.Time
}
```

The engine evaluates every rule, picks the highest-confidence hypothesis, and
clamps confidence to `[0, 1]`. Below the threshold it reports
`Insufficient evidence` while keeping the candidate's evidence for
explainability.

Confidence is assembled from explainable components (`score.go`):

| Component | Value |
| --- | --- |
| Strong evidence | +0.40 |
| Supporting evidence | +0.20 |
| Temporal correlation | +0.15 |
| Downstream impact | +0.15 |
| Anomaly confirmation | +0.10 |
| Contradiction | −0.20 |

## Rule System

Built-in rules live in `internal/diagnosis/rules/`:

| Rule | Severity | Priority behavior |
| --- | --- | --- |
| database_unavailable | critical | upstream; outranks 5xx |
| redis_unavailable | critical | upstream |
| out_of_memory | critical | preferred over crash |
| mq_unavailable | critical | upstream |
| network_partition | critical | upstream |
| disk_full | critical | independent strong signal |
| certificate_expired | high | independent strong signal |
| connection_pool_exhausted | high | downgraded when DB is down |
| cpu_saturation | high | weighs timeline trend |
| deadlock | high | independent strong signal |
| application_crash | high | downgraded when OOM present |
| http_5xx_spike | high | symptom; never outranks upstream |
| connection_timeout | medium | symptom-type |
| slow_query | medium | symptom-type |

Custom rules (`rules/custom.go`) implement the same interface from YAML
configuration and compete on equal footing.

### Adding a Diagnosis Rule

1. Create `internal/diagnosis/rules/<name>.go` implementing `DiagnosisRule`.
2. Build confidence from the shared components in `score.go`; never hard-code
   a value.
3. Return `nil` when there is no evidence.
4. Provide read-only, low-risk recommendations.
5. Handle priority: upstream rules join `upstreamDown` in `http.go`; symptoms
   start from a supporting weight.
6. Register in `RegisterDefaultRules` (`registry.go`).
7. Add a fixture under `testdata/incidents/` plus tests, and wire the fixture
   into `TestIncidentFixturesDiagnosis`.

**A diagnosis must be evidence-based.** Never: single keyword → high
confidence, filename-based diagnosis, or random confidence.

## Configuration

`internal/config` merges sources with increasing priority:

1. Built-in defaults (`Default()`)
2. `.faultlens.yaml` (project)
3. `~/.config/faultlens/config.yaml` (user)
4. `--config <file>` (explicit)

Merge rules: zero values never overwrite; maps merge per key; pointer booleans
allow explicit `false`. Unknown keys are tolerated for forward compatibility.
`Validate()` checks anomaly thresholds, rule thresholds and custom rule
fields. The anomaly detector reads its thresholds from the resolved config.

## Output Layer

`internal/output` renders `engine.Result`:

| Renderer | Purpose |
| --- | --- |
| `terminal.go` | human-readable report (4 views: report/errors/timeline/incident) |
| `json.go` | stable schema for CI and tooling |
| `markdown.go` | issues, postmortems, PR comments |
| `html.go` | self-contained offline report with inline SVG timeline |
| `diff.go` | compares two `engine.Result` JSON documents |

The JSON schema is fixed by struct tags on `engine.Result` and its nested
types — a stable contract for GitHub Actions and other consumers.

## CI/CD Integration

- `.github/workflows/test.yml` — CI (gofmt + vet + `go test -short`).
- `.github/workflows/ci-analysis.yml` — example workflow producing JSON + HTML
  artifacts and a job summary.
- `.github/workflows/release.yml` — tag-triggered multi-platform build
  (linux/darwin/windows × amd64/arm64), checksums, GitHub Release.
- `.github/actions/faultlens/action.yml` — composite action usable in any
  workflow: builds the CLI, runs the analysis, writes the report.

The `faultlens` CLI is CI-friendly by contract: stable stdout/stderr/exit code
and a stable JSON schema.

## Extension Points

- **New log format** → `internal/parser` (see [Adding a Parser](#adding-a-parser)).
- **New diagnosis rule** → `internal/diagnosis/rules` (see
  [Adding a Diagnosis Rule](#adding-a-diagnosis-rule)).
- **Custom rule without code** → `custom_rules` in configuration (see the
  [README Custom Rules section](../README.md#custom-rules)).
- **New configuration option** → `internal/config` (field + default + merge +
  validation).
- **New report format** → `internal/output` renderers.