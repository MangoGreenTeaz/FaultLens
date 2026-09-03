# Changelog

All notable changes to FaultLens are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-09-03

### Added

- **V1 core pipeline** — streaming input → parser → normalize → grouping →
  timeline → anomaly → evidence-based diagnosis → report
- **Parsers (9)** — plain, JSON, Java/Spring Boot (stack-trace aggregation),
  Nginx, Apache, Python, Syslog (RFC 3164 + 5424), Docker JSON, Kubernetes;
  content-based auto detection
- **Diagnosis rules (14)** — database, redis, oom, timeout, http 5xx, crash,
  disk full, certificate expired, mq, connection pool, network partition,
  cpu saturation, slow query, deadlock
- **Configuration system** — built-in/project/user/`--config` merge,
  `faultlens config init|show|validate`
- **Custom rule engine** — YAML-defined diagnosis rules without source changes
- **Multi-file input** — glob, directory recursion, `--exclude`, source
  tracking
- **HTML report** — single-file, offline, inline SVG timeline
- **`faultlens diff`** — compare two JSON reports (added/removed/changed
  groups, diagnosis changes)
- **GitHub Actions** — CI workflow plus official composite action and job
  summary
- **Performance** — streaming HTTP 5xx stats (500MB processed with ~2MB
  resident heap), benchmarks and a 500MB acceptance check
- **Release engineering** — module path unified to
  `github.com/MangoGreenTeaz/FaultLens`, multi-platform release workflow
  (linux/darwin/windows × amd64/arm64), checksums, ldflags version injection
- **OSS infrastructure** — bilingual README, CONTRIBUTING, SECURITY,
  CHANGELOG, issue/PR templates, examples

### Changed

- Module path: `github.com/faultlens/faultlens` → `github.com/MangoGreenTeaz/FaultLens`
- Default version: `0.2.0-dev`

## [Unreleased]

### Planned

- V3 ideas only (real-time tail, daemon mode, web UI, rule learning) — see
  `plan-v2.md`