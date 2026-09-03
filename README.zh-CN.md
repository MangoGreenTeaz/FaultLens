# FaultLens

> See beyond the error.

**🌐 语言:** [English](README.md) | [简体中文](README.zh-CN.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://go.dev/dl/)
[![CI](https://github.com/MangoGreenTeaz/FaultLens/actions/workflows/test.yml/badge.svg)](https://github.com/MangoGreenTeaz/FaultLens/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MangoGreenTeaz/FaultLens)](https://goreportcard.com/report/github.com/MangoGreenTeaz/FaultLens)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

FaultLens 是一个 **local-first、offline-first** 的日志事故诊断 CLI。它能识别
错误模式、发现异常时间点、分析错误之间的关系,并基于透明、可解释的规则推断
最可能的 Root Cause——**无需网络、无需数据库、无需 AI API**。

```bash
faultlens app.log
# 或者
cat app.log | faultlens
```

## 目录

- [特性](#特性)
- [安装](#安装)
- [快速开始](#快速开始)
- [支持的日志](#支持的日志)
- [示例](#示例)
- [工作原理](#工作原理)
- [架构](#架构)
- [CLI 参考](#cli-参考)
- [输出格式](#输出格式)
- [测试](#测试)
- [贡献](#贡献)
- [许可](#许可)

## ✨ 特性

- 🔍 **统一解析** — 纯文本、JSON(支持字段别名)、Java/Spring Boot(多行堆栈
  跟踪合并为单个事件)、Nginx access/error 日志;格式按内容自动检测。
- 🧹 **动态值归一化** — IP、端口、UUID、数字、时间戳、URL、路径、Hex ID 变为
  稳定占位符,同时保留 HTTP 状态码等有诊断意义的数值。
- 🗂️ **错误聚类** — 语义相同的错误按稳定 SHA-256 指纹聚为一组,按出现次数
  排序。
- 📈 **异常检测** — 按分钟分桶,与可解释的基线(均值 + 标准差、z-score)对比。
  无机器学习,完全透明。
- 🧠 **可解释诊断** — 每个 Root Cause 都附带证据、规则和可追溯到日志行的置信
  度。证据不足时输出 `Insufficient evidence`,而不是强行猜测。
- 📦 **多种报告格式** — Terminal 文本、稳定 JSON(供 CI 与工具消费)、Markdown
  (适合 Issue、Postmortem、PR 评论)。
- 🔒 **本地优先 & 离线** — 一切都在本机运行,无遥测、无外部调用。

## 📦 安装

**环境要求:** [Go 1.22+](https://go.dev/dl/)

从源码构建(模块发布前推荐方式):

```bash
git clone https://github.com/MangoGreenTeaz/FaultLens.git
cd FaultLens
go build -o faultlens ./cmd/faultlens
```

模块发布到 `faultlens` org 后,可直接安装:

```bash
go install github.com/faultlens/faultlens/cmd/faultlens@latest
```

验证安装:

```bash
faultlens version
```

## 🚀 快速开始

分析日志文件:

```bash
faultlens app.log
```

分析特定维度:

```bash
faultlens errors app.log     # 错误聚类
faultlens timeline app.log   # 按分钟时间线 + 异常
faultlens incident app.log   # Root Cause 诊断
```

从 stdin 管道输入:

```bash
cat app.log | faultlens
docker logs some-container | faultlens incident
```

按时间窗口过滤:

```bash
faultlens incident app.log --from 2026-08-31T14:00:00Z --to 2026-08-31T15:00:00Z
```

## 📊 支持的日志

| 格式 | 示例 | 说明 |
| --- | --- | --- |
| 纯文本 | `2026-08-31 14:32:01 ERROR database connection failed` | `timestamp + level + message` |
| JSON | `{"timestamp":"2026-08-31T14:32:01Z","level":"ERROR","message":"..."}` | JSONL;别名:`timestamp/time/ts`、`level/severity`、`message/msg`、`service/service_name` |
| Java / Spring Boot | `2026-08-31T14:32:01.123+08:00 ERROR ...` + 堆栈跟踪 | 堆栈跟踪(含 `Caused by:`、`Suppressed:`)聚合为单个事件 |
| Nginx | `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 500 123` | access + error 日志;status/method/path 提取到字段 |

使用 `--format auto|plain|json|java|nginx` 可覆盖内容检测。

## 📝 示例

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

## ⚙️ 工作原理

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

1. **解析** 原始日志行为统一的 `LogEvent` 模型。
2. **归一化** 动态值,使 `Connection refused 10.0.0.1:3306` 与
   `Connection refused 10.0.0.2:3306` 变成同一形状。
3. **聚类** 按指纹归并相同错误并计数。
4. **构建** 按分钟分桶,**检测** 与可解释基线相比的异常尖峰。
5. **诊断** 对每个候选 Root Cause 以证据与置信度打分;症状(HTTP 5xx)永远
   不会压过其上游根因。
6. **报告** 以你选择的格式输出。

## 🏗️ 架构

```
cmd/faultlens/       CLI 入口(Cobra)
internal/engine/     分析管线(input → … → diagnosis)
internal/model/      共享数据模型(LogEvent、Diagnosis 等)
internal/input/      文件与 stdin 流式输入
internal/parser/     plain / json / java / nginx / auto 解析器
internal/normalize/  错误归一化
internal/grouping/   错误聚类 + 指纹
internal/timeline/   按分钟时间桶
internal/anomaly/    z-score 异常检测
internal/diagnosis/  Root Cause 引擎 + 6 条规则
internal/output/     Terminal / JSON / Markdown 渲染器
testdata/            样例日志与事故 fixtures
```

核心分析包与 CLI 解耦,管线未来可复用于 GitHub Actions 或 API 集成。

## 💻 CLI 参考

```bash
faultlens <file>          # 完整分析报告(默认)
faultlens errors <file>   # 错误聚类
faultlens timeline <file> # 按分钟时间线 + 异常
faultlens incident <file> # Root Cause 诊断
faultlens diff a.json b.json # 对比两份 JSON 报告
faultlens version         # 打印版本号
```

| Flag | 取值 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--format` | `auto`、`plain`、`json`、`java`、`nginx` | `auto` | 日志格式 |
| `--output` | `terminal`、`json`、`markdown` | `terminal` | 报告格式 |
| `--from` | RFC 3339 时间 | — | 只分析该时间点之后的事件 |
| `--to` | RFC 3339 时间 | — | 只分析该时间点之前的事件 |

## 🖨️ 输出格式

- **terminal** — 人类可读的报告(默认)。
- **json** — 供工具与 CI 消费的稳定 schema:
  `{ "summary", "error_groups", "timeline", "anomalies", "diagnosis" }`。
- **html** — 单文件、完全离线的报告,含内联 SVG 错误时间线;适合 CI
  artifact 与事故分享。
- **markdown** — 表格与标题,适合 GitHub Issue、Postmortem、PR 评论。

## 🤖 GitHub Actions

FaultLens 可以自然融入 CI/CD 工作流:JSON 供机器消费,HTML 作为可分享的
artifact。

分析日志并产出两种 artifact:

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

或者使用本仓库内置的官方 composite action:

```yaml
- uses: ./.github/actions/faultlens
  with:
    log-file: testdata/incidents/mysql-outage.log
    output: html
    output-file: faultlens-report.html
```

完整的可运行示例见
[`.github/workflows/ci-analysis.yml`](.github/workflows/ci-analysis.yml)——它会
分析 fixture、上传 JSON + HTML artifacts,并写入 job summary。

## 🧪 测试

```bash
go test ./...                                    # 每个包的单元测试
go test ./internal/engine/ -run TestIncidentFixtures -v   # 端到端测试
go vet ./...
```

每个核心包都有 table-driven 单元测试;`testdata/incidents/*` fixtures 驱动
完整管线集成测试——例如 `mysql-outage.log` 必须诊断为 `Database
unavailable`,而不是 HTTP 5xx 症状。

## 🤝 贡献

欢迎贡献!请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。它涵盖如何新增
parser、如何新增 diagnosis rule、如何提交 PR。

## 📄 许可

FaultLens 以 [MIT 许可证](LICENSE) 发布。

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

向 FaultLens 贡献代码即表示你同意你的贡献同样以 MIT 许可证授权。