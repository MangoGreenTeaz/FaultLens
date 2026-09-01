# Contributing to FaultLens

Thanks for your interest in contributing! FaultLens is a local-first,
offline-first log incident diagnosis CLI. We value **correct, explainable,
testable, and maintainable** code over feature count.

## Table of Contents

- [Getting Started](#getting-started)
- [How to Add a Parser](#how-to-add-a-parser)
- [How to Add a Diagnosis Rule](#how-to-add-a-diagnosis-rule)
- [How to Add Test Data](#how-to-add-test-data)
- [Submitting a Pull Request](#submitting-a-pull-request)
- [Code of Conduct](#code-of-conduct)

## Getting Started

Clone the repository:

```bash
git clone https://github.com/faultlens/faultlens.git
cd faultlens
```

Build and verify:

```bash
go build ./...
go test ./...
go vet ./...
```

All code must be `gofmt`-clean. Run the checks before submitting:

```bash
gofmt -l .
go vet ./...
go test ./...
```

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

2. **Register it in auto detection**: add your parser to the priority chain in
   `detectFormat` in `internal/parser/auto.go`.

3. **Add table-driven tests** in `internal/parser/your_format_test.go`,
   covering normal input, malformed input, and edge cases. If your format has
   multi-line events, test aggregation and `Flush`.

4. If your format introduces new structured fields, document them in the
   `LogEvent.Fields` contract in `internal/model/event.go`.

## How to Add a Diagnosis Rule

Rules live in `internal/diagnosis/rules/`. Each rule evaluates one root-cause
hypothesis against the pre-computed `diagnosis.DiagnosisContext` and returns a
`model.Diagnosis`.

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
   Never hard-code a confidence value. Rules must clamp to `[0, 1]` (the
   engine does this automatically).

3. **Return `nil` when your rule finds no evidence** so the engine can skip it.

4. **Provide read-only, low-risk recommendations** — never suggest
   kill/restart/delete/flush/drop style commands.

5. **Register the rule** in `RegisterDefaultRules` in
   `internal/diagnosis/rules/registry.go`. Order defines tie-breaking
   priority.

6. **Add tests** in `internal/diagnosis/rules/` covering: strong evidence,
   weak/insufficient evidence, and interaction with existing rules (e.g. a
   symptom rule that must downgrade when an upstream cause exists).

## How to Add Test Data

- Format samples go in `testdata/<format>/` (e.g. `testdata/plain/basic.log`).
- Incident scenarios go in `testdata/incidents/` and must simulate a realistic
  causal chain (e.g. `database failure → connection refused → app errors →
  HTTP 500`).
- End-to-end tests in `internal/engine/integration_test.go` assert the
  expected diagnosis for each incident fixture. **Diagnosis must come from log
  evidence, never from the fixture file name.**

## Submitting a Pull Request

1. Fork the repository and create a branch: `git checkout -b feat/my-change`.
2. Make focused, small changes. Keep each PR about one thing.
3. Run `gofmt -l .`, `go vet ./...`, and `go test ./...` — all must pass.
4. Commit with a clear message following the existing style
   (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`).
5. Open the PR against `main`. Describe what changed and why, and reference
   the issue if one exists.

## Code of Conduct

Be respectful and constructive. We are all here to learn and build together.