# Contributing to FaultLens

Thanks for your interest in contributing! FaultLens is a local-first,
offline-first log incident diagnosis CLI. We value **correct, explainable,
testable, and maintainable** code over feature count.

## Table of Contents

- [Getting Started](#getting-started)
- [Reporting Bugs](#reporting-bugs)
- [Good First Issues](#good-first-issues)
- [How to Add a Parser](#how-to-add-a-parser)
- [How to Add a Diagnosis Rule](#how-to-add-a-diagnosis-rule)
- [How to Add a Configuration Option](#how-to-add-a-configuration-option)
- [How to Add Test Data](#how-to-add-test-data)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Code of Conduct](#code-of-conduct)

## Getting Started

Install the latest release:

```bash
go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest
```

Or build from source:

```bash
git clone https://github.com/MangoGreenTeaz/FaultLens.git
cd FaultLens
go build ./...
```

Run the checks before submitting:

```bash
gofmt -l .
go vet ./...
go test ./...        # add -short to skip the 500MB performance check
```

## Reporting Bugs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include
`faultlens version`, your OS, the input format, expected vs actual behavior,
and a minimal reproduction log (redact secrets). Feature requests, parser
requests and rule requests have their own templates.

## Good First Issues

If you are new here, these are real, self-contained tasks to start with.
Open an issue describing which one you are taking (or just open a PR):

- **Add a parser fixture** — a small realistic log sample under
  `testdata/<format>/` for a format we already parse.
- **Add a malformed-log test** — a table-driven case for a parser covering a
  malformed or boundary line.
- **Add a diagnosis rule test** — a strong/weak/no-evidence case for a built-in
  rule.
- **Improve a CLI error message** — make an error clearer for end users.
- **Improve documentation** — fix a README section or translate examples.

## How to Add a Parser

Parsers live in `internal/parser/`. A parser converts raw log lines into the
unified `model.LogEvent`.

1. **Implement the `Parser` interface** in `internal/parser/parser.go`:

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

   - `Parse` is stateful: feed one line, return any completed events. Use it
     for formats with multi-line events (e.g. Java stack traces).
   - `CanParse` must only return `true` for lines that *start* an event in
     your format (never for continuation lines).
   - `Issues` counts lines your parser could not convert into events.

2. **Register it**: add your parser to `detectFormat` in
   `internal/parser/auto.go` (priority chain) and to `newParser` in
   `internal/engine/engine.go` (`--format` flag). Mind the detection order:
   more specific formats first.

3. **Add fixtures** in `testdata/<format>/` and **table-driven tests** in
   `internal/parser/your_format_test.go` covering normal, malformed, and edge
   cases (multi-line aggregation and `Flush` if applicable).

4. If your format introduces new structured fields, document them in
   `internal/model/event.go`.

## How to Add a Diagnosis Rule

Rules live in `internal/diagnosis/rules/`. First check whether a
[custom rule](examples/disk-full/) already covers your case — users can define
rules in YAML without code. Built-in rules are for widely applicable signals.

1. **Implement `diagnosis.DiagnosisRule`**:

   ```go
   type DiagnosisRule interface {
       ID() string
       Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis
   }
   ```

2. **Build confidence from explainable components** in
   `internal/diagnosis/score.go` (`ScoreStrong`, `ScoreSupporting`,
   `ScoreTemporal`, `ScoreDownstream`, `ScoreAnomaly`, `ScoreContradict`).
   Never hard-code a confidence value.

3. **Return `nil` when your rule finds no evidence** so the engine skips it.

4. **Provide read-only, low-risk recommendations** — never suggest
   kill/restart/delete/flush/drop style commands.

5. **Handle priority**: if your rule is an upstream failure (mq, network…),
   add it to `upstreamDown` in `http.go` so HTTP 5xx downgrades correctly;
   if it is a symptom, start from a supporting weight.

6. **Register** in `RegisterDefaultRules` in
   `internal/diagnosis/rules/registry.go` (order defines tie-breaking).

7. **Add tests** (strong/weak/no evidence) and an incident fixture under
   `testdata/incidents/`; add the fixture to
   `internal/engine/integration_test.go`.

## How to Add a Configuration Option

Configuration lives in `internal/config/`.

1. **Add the field** to the matching struct in `internal/config/config.go`
   with a `yaml:"..."` tag.
2. **Add a default** in `Default()`.
3. **Wire it into the merge** in `internal/config/load.go` (zero values must
   not overwrite; use a pointer bool if `false` is meaningful).
4. **Validate it** in `internal/config/validate.go`.
5. **Consume it** in the relevant package (e.g. the anomaly detector reads
   `cfg.Anomaly.*`).
6. **Add tests** for the default, merge, and validation of the new option.

## How to Add Test Data

- Format samples go in `testdata/<format>/`.
- Incident scenarios go in `testdata/incidents/` and must simulate a realistic
  causal chain (e.g. `database failure → connection refused → app errors →
  HTTP 500`).
- End-to-end tests in `internal/engine/integration_test.go` assert the
  expected diagnosis. **Diagnosis must come from log evidence, never from the
  fixture file name.**

## Submitting a Pull Request

1. Fork the repository and create a branch: `git checkout -b feat/my-change`.
2. Make focused, small changes. Keep each PR about one thing.
3. Run `gofmt -l .`, `go vet ./...`, and `go test ./...` — all must pass.
4. Fill in the [PR template](.github/PULL_REQUEST_TEMPLATE.md).
5. Commit with a clear message following the existing style
   (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `perf:`).
6. Open the PR against `main`.

## Code of Conduct

Be respectful and constructive. We are all here to learn and build together.