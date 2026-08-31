# FaultLens

> See beyond the error. — 看见错误背后的真相。

**🌐 [English](#english) | [简体中文](#chinese)**

---

<div id="english"></div>

## English

FaultLens is a **local-first, offline-first** log incident diagnosis CLI. It
identifies error patterns, detects anomalous time points, analyzes the
relationships between errors, and infers the most likely root cause from
transparent, explainable rules — no network, no database, no AI API required.

> **Status: WIP** — V1 is under active development. Command behavior and output
> formats below are the target design and may change until the V1 release.

### Features

- Detect error patterns in plain text, JSON, Java/Spring Boot, and Nginx logs
- Normalize dynamic values (IPs, ports, UUIDs, numbers, timestamps, URLs…)
- Cluster errors by stable fingerprints
- Build per-minute timelines and detect anomalies with explainable baselines
- Diagnose the most likely root cause with evidence and confidence
- Output reports as terminal text, JSON, or Markdown

*Full feature list is being finalized in V1.*

### Installation

> TODO: documented once a stable release is available.

```bash
go install github.com/faultlens/faultlens/cmd/faultlens@latest
```

### Quick Start

```bash
faultlens app.log
```

Or pipe logs from stdin:

```bash
cat app.log | faultlens
```

### Supported Logs

- Plain text (timestamp + level + message)
- JSON (JSONL)
- Java / Spring Boot (including stack traces)
- Nginx access and error logs

### Example

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

### How It Works

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

Every diagnosis is backed by evidence, a rule, and a confidence score that can
be traced back to the log lines that produced it. If the evidence is not
strong enough, FaultLens reports `Insufficient evidence` instead of guessing.

### Architecture

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

### CLI Reference

> Commands below are the V1 target. Only `version` is implemented so far.

- `faultlens <file>` — full analysis report (default)
- `faultlens errors <file>` — error clustering
- `faultlens timeline <file>` — timeline analysis
- `faultlens incident <file>` — incident diagnosis
- `faultlens version` — print version

### Output Formats

- `terminal` — human-readable report
- `json` — stable schema for tooling and CI
- `markdown` — for issues, postmortems, and PR comments

### Development

```bash
go build ./...
go test ./...
go vet ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

### Testing

> TODO: describe the test layout and fixtures once populated.

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

### License

[Apache-2.0](LICENSE)

---

<div id="chinese"></div>

## 简体中文

FaultLens 是一个 **local-first、offline-first** 的日志事故诊断 CLI。它能识别
错误模式、发现异常时间点、分析错误之间的关系,并基于透明、可解释的规则推断
最可能的 Root Cause——无需网络、无需数据库、无需 AI API。

> **状态:开发中(WIP)** — V1 正在积极开发中。下文中的命令行为和输出格式属于
> 目标设计,在 V1 发布前可能发生变化。

### 特性

- 识别纯文本、JSON、Java/Spring Boot、Nginx 日志中的错误模式
- 归一化动态值(IP、端口、UUID、数字、时间戳、URL 等)
- 基于稳定指纹聚类错误
- 构建按分钟划分的时间线,用可解释的基线检测异常
- 基于证据与置信度诊断最可能的 Root Cause
- 输出 Terminal 文本、JSON 或 Markdown 报告

*V1 的完整功能清单仍在最终确定中。*

### 安装

> TODO:稳定版发布后再补充说明。

```bash
go install github.com/faultlens/faultlens/cmd/faultlens@latest
```

### 快速开始

```bash
faultlens app.log
```

或者从 stdin 管道输入日志:

```bash
cat app.log | faultlens
```

### 支持的日志

- 纯文本(timestamp + level + message)
- JSON(JSONL)
- Java / Spring Boot(包含堆栈跟踪)
- Nginx access 与 error 日志

### 示例

> 目标输出(V1 开发中):

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

### 工作原理

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

每一条诊断都由证据、规则和置信度支撑,可以追溯到产生它的日志行。当证据不足
时,FaultLens 会输出 `Insufficient evidence`,而不是强行猜测。

### 架构

```
cmd/faultlens/   CLI 入口(Cobra)
internal/model/  共享数据模型
internal/input/  文件与 stdin 输入
internal/parser/ 日志格式解析器
internal/normalize/ 错误归一化
internal/grouping/  错误聚类
internal/timeline/  时间桶分析
internal/anomaly/   异常检测
internal/diagnosis/ Root Cause 引擎与规则
internal/output/    Terminal / JSON / Markdown 渲染器
```

核心分析包与 CLI 解耦,未来可复用于 GitHub Actions 或 API 集成。

### CLI 参考

> 以下命令是 V1 的目标设计。目前仅实现了 `version`。

- `faultlens <file>` — 完整分析报告(默认)
- `faultlens errors <file>` — 错误聚类
- `faultlens timeline <file>` — 时间线分析
- `faultlens incident <file>` — 事故诊断
- `faultlens version` — 打印版本号

### 输出格式

- `terminal` — 人类可读的报告
- `json` — 供工具与 CI 消费的稳定 schema
- `markdown` — 适合 Issue、Postmortem、PR 评论

### 开发

```bash
go build ./...
go test ./...
go vet ./...
```

贡献指南见 [CONTRIBUTING.md](CONTRIBUTING.md)。

### 测试

> TODO:测试布局与 fixtures 就绪后补充说明。

### 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。

### 许可

[Apache-2.0](LICENSE)