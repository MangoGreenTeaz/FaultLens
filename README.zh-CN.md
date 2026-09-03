# FaultLens

> **面向开发者与 CI/CD 的 local-first、可解释日志事故诊断工具。**
>
> 将嘈杂的应用日志转化为结构化事故、基于证据的 Root Cause 假设与可执行的排查建议。
>
> **无云服务。无数据库。无遥测。无 LLM API。**

**🌐 语言:** [English](README.md) | [简体中文](README.zh-CN.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://go.dev/dl/)
[![CI](https://github.com/MangoGreenTeaz/FaultLens/actions/workflows/test.yml/badge.svg)](https://github.com/MangoGreenTeaz/FaultLens/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/MangoGreenTeaz/FaultLens)](https://github.com/MangoGreenTeaz/FaultLens/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MangoGreenTeaz/FaultLens)](https://goreportcard.com/report/github.com/MangoGreenTeaz/FaultLens)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

## 目录

- [演示](#演示)
- [为什么用 FaultLens](#为什么用-faultlens)
- [特性](#特性)
- [快速开始](#快速开始)
- [安装](#安装)
- [支持的日志格式](#支持的日志格式)
- [诊断规则](#诊断规则)
- [自定义规则](#自定义规则)
- [多文件分析](#多文件分析)
- [GitHub Actions](#github-actions)
- [HTML 报告](#html-报告)
- [Diff](#diff)
- [工作原理](#工作原理)
- [架构](#架构)
- [示例](#示例)
- [开发](#开发)
- [贡献](#贡献)
- [安全](#安全)
- [路线图](#路线图)
- [许可](#许可)

## 演示

```bash
faultlens examples/basic/app.log
```

```text
FaultLens
────────────────────────────────────────────

Log Summary

Events:       7
Errors:       3
Warnings:     1
Time Range:   10:00:01 - 10:00:07
Format:       java

────────────────────────────────────────────

Top Error Groups

1. connection refused <IP>:<PORT>
   1 occurrences

2. MySQL connection failed
   1 occurrences

3. HTTP 500 Internal Server Error
   1 occurrences

────────────────────────────────────────────

Diagnosis

ROOT CAUSE
Database unavailable

Confidence
0.70

Severity
critical

Evidence
10:00:06  ERROR_PATTERN
database-related errors detected

10:00:04  ERROR_PATTERN
connection failures detected

10:00:07  DOWNSTREAM_IMPACT
HTTP 5xx errors observed downstream

10:00:07  TIMELINE_CORRELATION
database errors preceded HTTP 5xx spike

Recommendations
1. Check MySQL availability
2. Check database connection limit
3. Check recent database restart
4. Check network connectivity between application and database
```

## 为什么用 FaultLens

日志文件中已经包含了事故的全部故事——但它们嘈杂、非结构化、充满重复错误。
FaultLens 将它们转化为可解释的诊断:

- **本地优先 & 离线** — 一切在本机运行,数据不离机。
- **可解释** — 每条诊断都附带证据、规则和可追溯到日志行的置信度。
- **快速** — 流式管线;500MB 日志几分钟处理完,常驻堆仅约 2MB。
- **CI/CD 友好** — 稳定 JSON 输出与 HTML artifact 可融入任意管线。
- **可扩展** — 用 YAML 定义诊断规则,无需修改源码。

## 特性

- 🔍 **9 种日志格式解析器** — 纯文本、JSON、Java/Spring Boot(多行堆栈跟踪
  合并为单个事件)、Nginx、Apache、Python、Syslog(RFC 3164 + 5424)、Docker
  JSON、Kubernetes;基于内容自动检测。
- 🧹 **动态值归一化** — IP、端口、UUID、数字、时间戳、URL、路径、Hex ID 变为
  稳定占位符;HTTP 状态码保留诊断意义。
- 🗂️ **错误聚类** — 相同错误按 SHA-256 指纹合并,按出现次数排序。
- 📈 **异常检测** — 按分钟分桶,与可解释基线(均值 + z-score)对比,无机器学习。
- 🧠 **14 条内置诊断规则** — 数据库、Redis、OOM、超时、HTTP 5xx、崩溃、磁盘
  满、证书、消息队列、连接池、网络分区、CPU 饱和、慢查询、死锁。
- ⚙️ **YAML 自定义规则** — 无需编写 Go 即可扩展诊断。
- 📁 **多文件分析** — 目录、glob、`--exclude`。
- 🖥️ **报告格式** — terminal、JSON、Markdown、含 SVG 时间线的离线 HTML。
- 🔀 **Diff** — 对比两次分析结果(部署前后)。
- 🤖 **GitHub Actions** — 在 CI 中分析日志,发布 JSON + HTML artifacts。

## 快速开始

```bash
go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest

faultlens app.log                # 完整报告
faultlens errors app.log         # 错误聚类
faultlens timeline app.log       # 时间线 + 异常
faultlens incident app.log       # Root Cause 诊断
```

或从 stdin 管道输入:

```bash
cat app.log | faultlens
docker logs some-container | faultlens incident
```

## 安装

**环境要求:** [Go 1.22+](https://go.dev/dl/)

安装最新发布版:

```bash
go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest
```

预编译二进制见 [Releases](https://github.com/MangoGreenTeaz/FaultLens/releases)
页面:Linux、macOS、Windows × amd64、arm64(Windows 直接提供 `.exe`)。

或从源码构建:

```bash
git clone https://github.com/MangoGreenTeaz/FaultLens.git
cd FaultLens
go build -o faultlens ./cmd/faultlens
```

验证:

```bash
faultlens version
```

## 支持的日志格式

| 格式 | 状态 | 说明 |
| --- | --- | --- |
| 纯文本 | Stable | `timestamp + level + message` |
| JSON | Stable | JSONL;字段别名(`time`/`ts`、`severity`、`msg` 等) |
| Java / Spring Boot | Stable | 多行堆栈跟踪合并为单个事件 |
| Nginx | Stable | access + error 日志;status/method/path 提取 |
| Apache | Stable | common / combined / vhost-combined |
| Python | Stable | 标准 `logging` 格式(逗号小数秒) |
| Syslog | Stable | RFC 3164 + RFC 5424;PRI → severity 映射 |
| Docker JSON | Stable | `{"log","stream","time"}` |
| Kubernetes | Stable | Kubelet 容器日志(`ts stdout F msg`) |

可用 `--format auto|plain|json|java|nginx|apache|python|syslog|docker|kubernetes`
覆盖内容检测。

## 诊断规则

14 条内置规则,每条按证据 → 置信度打分:

| 规则 ID | Root Cause | Severity | 关键证据 |
| --- | --- | --- | --- |
| `database_unavailable` | Database unavailable | critical | mysql/postgres/sql/jdbc + 连接失败 + 5xx 下游 |
| `redis_unavailable` | Redis unavailable | critical | redis/cache 错误 + 连接失败 |
| `out_of_memory` | Out of memory | critical | OOM 关键词 + 堆栈跟踪 |
| `mq_unavailable` | Message queue unavailable | critical | rabbitmq/amqp/kafka + 5xx 下游 |
| `network_partition` | Network partition | critical | 网络/主机不可达 + 5xx 下游 |
| `disk_full` | Disk full | critical | 磁盘空间不足、写入错误 |
| `certificate_expired` | Certificate expired | high | x509、SSL 握手失败 |
| `connection_pool_exhausted` | Connection pool exhausted | high | 连接池耗尽;DB 故障时降级 |
| `cpu_saturation` | CPU saturation | high | CPU 使用率 + 时间序列趋势 |
| `deadlock` | Deadlock detected | high | 死锁、锁超时 |
| `application_crash` | Application crash | high | fatal/panic;OOM 存在时降级 |
| `http_5xx_spike` | HTTP 5xx spike | high | 5xx(症状,不压过上游根因) |
| `connection_timeout` | Connection timeout | medium | 读/套接字/连接超时 |
| `slow_query` | Slow query | medium | 慢查询(症状型) |

## 自定义规则

无需修改源码即可扩展诊断——用 YAML 定义规则:

```yaml
custom_rules:
  - id: database_connection_failure
    root_cause: "Database connection failure"
    severity: high
    keywords:
      - connection refused
      - database unavailable
    strong_weight: 0.40
    supporting_keywords:
      - network unreachable
    supporting_weight: 0.20
    enable_downstream: true
    recommendations:
      - "Check database availability"
      - "Check network connectivity"
```

字段说明:

| 字段 | 说明 |
| --- | --- |
| `id` | 唯一规则标识(不得与内置规则冲突) |
| `root_cause` | 报告中展示的诊断标签 |
| `severity` | `low` / `medium` / `high` / `critical` |
| `keywords` | 强证据关键词(至少 1 个) |
| `strong_weight` | 强证据命中时加的置信度(默认 0.40) |
| `supporting_keywords` | 可选的支撑证据关键词 |
| `supporting_weight` | 支撑证据命中时加的置信度(默认 0.20) |
| `enable_downstream` | 追加下游 HTTP 5xx + 时序证据 |
| `recommendations` | 只读、低风险的排查建议 |

自定义规则与内置规则在同一证据评分引擎中竞争。管理配置:

```bash
faultlens config init       # 生成 .faultlens.yaml
faultlens config show       # 打印合并后的生效配置
faultlens config validate   # 校验配置文件
```

配置来源优先级(低 → 高):内置默认值 → `.faultlens.yaml`(项目) →
`~/.config/faultlens/config.yaml`(用户)→ `--config <file>`。

可运行示例见 [examples/custom-rule](examples/custom-rule/)。

## 多文件分析

将多个日志作为一次事故分析:

```bash
faultlens logs/                     # 递归目录(*.log、*.jsonl)
faultlens 'logs/**/*.log'           # glob
faultlens app.log database.log nginx.log   # 显式文件
faultlens logs/ --exclude '*.debug.log'    # 排除模式
```

所有文件的事件合并为单一时间线、聚类与诊断;每个错误示例保留来源文件。

## GitHub Actions

在 CI 中运行 FaultLens 并发布报告 artifacts:

```yaml
name: FaultLens

on:
  workflow_run:
    workflows: ["Tests"]
    types: [completed]

jobs:
  diagnose:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install FaultLens
        run: go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest

      - name: Analyze logs
        run: |
          faultlens ./logs/app.log --output json > faultlens.json
          faultlens ./logs/app.log --output html -o faultlens-report.html

      - name: Upload report
        uses: actions/upload-artifact@v4
        with:
          name: faultlens-report
          path: |
            faultlens.json
            faultlens-report.html
```

仓库还内置官方 composite action:

```yaml
- uses: ./.github/actions/faultlens
  with:
    log-file: testdata/incidents/mysql-outage.log
    output: html
    output-file: faultlens-report.html
```

| Input | 说明 | 默认值 |
| --- | --- | --- |
| `log-file` | 要分析的日志路径 | 必填 |
| `output` | 报告格式:`json`、`html`、`terminal` | `html` |
| `output-file` | 报告输出位置 | `faultlens-report` |

composite action 在 GitHub Actions 内直接运行 FaultLens,无需托管服务。完整可
运行 workflow 见 [`.github/workflows/ci-analysis.yml`](.github/workflows/ci-analysis.yml)。

## HTML 报告

生成自包含的离线事故报告:

```bash
faultlens app.log --output html -o report.html
```

- **无 CDN、无外部 JavaScript、无后端、无网络依赖**
- 内联 CSS + 内联 SVG 错误时间线(异常桶高亮)
- 区段:summary、timeline、error groups、diagnosis、evidence、recommendations、
  source files

适合 CI artifact、事故分享与 postmortem。

## Diff

对比两次分析结果:

```bash
faultlens incident app.log --output json -o before.json
# ... 部署或修改日志 ...
faultlens incident app.log --output json -o after.json
faultlens diff before.json after.json
```

展示新增/消失/变化的错误组、诊断变化与置信度变化——非常适合定位部署回归。

## 工作原理

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

## 架构

```
cmd/faultlens/       CLI 入口(Cobra)
internal/engine/     分析管线(input → … → diagnosis)
internal/model/      共享数据模型(LogEvent、Diagnosis 等)
internal/input/      文件与 stdin 流式输入
internal/parser/     plain / json / java / nginx / apache / python / syslog /
                     docker / kubernetes 解析器
internal/normalize/  错误归一化
internal/grouping/   错误聚类 + 指纹
internal/timeline/   按分钟时间桶
internal/anomaly/    z-score 异常检测
internal/diagnosis/  Root Cause 引擎 + 内置/自定义规则
internal/output/     terminal / JSON / Markdown / HTML 渲染器
internal/config/     配置加载、合并与校验
testdata/            样例日志与事故 fixtures
examples/            可运行的端到端示例
```

核心分析包与 CLI 解耦,管线未来可复用于 GitHub Actions 或 API 集成。完整设计
见 [docs/architecture.md](docs/architecture.md)。

## 示例

每个示例都可运行,输出来自真实 CLI:

| 示例 | 展示 |
| --- | --- |
| [basic](examples/basic/) | 最小工作流 |
| [database-outage](examples/database-outage/) | 事故链 → Database unavailable |
| [network-partition](examples/network-partition/) | 网络根因 |
| [disk-full](examples/disk-full/) | 磁盘耗尽的自定义规则 |
| [custom-rule](examples/custom-rule/) | 从零定义自定义规则 |
| [ci](examples/ci/) | GitHub Actions workflow |

## 开发

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## 贡献

欢迎贡献——解析器、诊断规则、fixtures 与文档。参见
[CONTRIBUTING.md](CONTRIBUTING.md),包含 **good first contributions**。

## 安全

漏洞上报方式见 [SECURITY.md](SECURITY.md)。

## 路线图

V3 想法记录(未实现)见 [plan-v2.md](plan-v2.md):实时 tail、daemon 模式、
Web UI、规则学习、跨主机时间线对齐。

## 许可

FaultLens 以 [MIT 许可证](LICENSE) 发布。

向 FaultLens 贡献代码即表示你同意你的贡献同样以 MIT 许可证授权。