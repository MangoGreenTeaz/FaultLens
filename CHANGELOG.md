# Changelog

All notable changes to FaultLens are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added (V1 development)

- **Phase 1 — Project bootstrap**
  - Go module, Cobra CLI skeleton, `version` command
  - README skeleton, GitHub Actions CI (gofmt + vet + test)
- **Phase 2 — Model + Input**
  - Unified `LogEvent` model with `LogLevel` (TRACE…FATAL + UNKNOWN)
  - Streaming file and stdin input with long-line tolerance
- **Phase 3 — Parser**
  - Plain text, JSON (field aliases), Java/Spring Boot (multi-line stack
    trace aggregation), Nginx access/error parsers
  - Content-based auto format detection (JSON → Java → Nginx → Plain)
- **Phase 4 — Normalize + Grouping**
  - Dynamic value normalization (IP, port, UUID, number, timestamp, URL,
    path, hex) with HTTP status-code protection against over-normalization
  - SHA-256 fingerprinting and error clustering
- **Phase 5 — Timeline + Anomaly**
  - Per-minute time buckets
  - Explainable z-score anomaly detection (baseline + increase multiple)
- **Phase 6 — Diagnosis**
  - Diagnosis engine with evidence-based confidence scoring
  - Six root-cause rules: database, redis, oom, timeout, http 5xx, crash
  - Symptom-vs-cause priority (OOM > crash; upstream > HTTP 5xx)
  - `Insufficient evidence` reporting below the confidence threshold
- **Phase 7 — Output + CLI**
  - Terminal, JSON and Markdown renderers (4 report views each)
  - Full CLI: `faultlens <file>`, `errors`, `timeline`, `incident`
  - Flags: `--format`, `--output`, `--from`, `--to`
- **Phase 8 — End-to-end tests**
  - Realistic format samples and incident fixtures under `testdata/`
  - Full-pipeline integration tests (MySQL/Redis/OOM/5xx/crash scenarios)
- **Phase 9 — Documentation**
  - Bilingual README, CONTRIBUTING, SECURITY, CHANGELOG