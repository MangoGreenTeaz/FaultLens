# FaultLens

> See beyond the error. — 看见错误背后的真相。

**🌐 语言:** [English](README.md) | [简体中文](README.zh-CN.md)

FaultLens 是一个 **local-first、offline-first** 的日志事故诊断 CLI。它能识别
错误模式、发现异常时间点、分析错误之间的关系,并基于透明、可解释的规则推断
最可能的 Root Cause——无需网络、无需数据库、无需 AI API。

## 特性

- 识别纯文本、JSON、Java/Spring Boot、Nginx 日志中的错误模式
- 归一化动态值(IP、端口、UUID、数字、时间戳、URL 等)
- 基于稳定 SHA-256 指纹聚类错误
- 构建按分钟划分的时间线,用可解释的基线检测异常
- 基于证据与置信度诊断最可能的 Root Cause
- 证据不足时输出 `Insufficient evidence`,而不是强行猜测
- 输出 Terminal 文本、JSON 或 Markdown 报告

## 安装

需要 [Go 1.22+](https://go.dev/dl/)。

```bash
go install github.com/faultlens/faultlens/cmd/faultlens@latest
```

或从源码构建:

```bash
git clone https://github.com/faultlens/faultlens.git
cd faultlens
go build ./cmd/faultlens
```

## 快速开始

```bash
faultlens app.log
```

或者从 stdin 管道输入日志:

```bash
cat app.log | faultlens
docker logs some-container | faultlens incident
```

## 支持的日志

- 纯文本(`timestamp + level + message`)
- JSON(JSONL),支持常见字段别名(`timestamp`/`time`/`ts`、`level`/`severity` 等)
- Java / Spring Boot,多行堆栈跟踪聚合成单个事件
- Nginx access 与 error 日志

格式根据内容自动检测;可用 `--format` 强制指定。

## 示例

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

## 工作原理

```
input → parser → normalize → grouping → timeline → anomaly → diagnosis → report
```

每一条诊断都由证据、规则和置信度支撑,可以追溯到产生它的日志行。置信度由
可解释的组件叠加而成(强证据、支撑证据、时序相关、下游影响、矛盾信号),并
限制在 `[0, 1]` 区间。

## 架构

```
cmd/faultlens/       CLI 入口(Cobra)
internal/engine/     分析管线(input → … → diagnosis)
internal/model/      共享数据模型
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

核心分析包与 CLI 解耦,未来可复用于 GitHub Actions 或 API 集成。

## CLI 参考

```bash
faultlens <file>          # 完整分析报告(默认)
faultlens errors <file>   # 错误聚类
faultlens timeline <file> # 按分钟时间线 + 异常
faultlens incident <file> # Root Cause 诊断
faultlens version         # 打印版本号
```

全局 flags:

```bash
--format auto|plain|json|java|nginx   # 日志格式(默认 auto)
--output terminal|json|markdown       # 报告格式(默认 terminal)
--from <RFC3339>                      # 只分析该时间点之后的事件
--to   <RFC3339>                      # 只分析该时间点之前的事件
```

## 输出格式

- `terminal` — 人类可读的报告
- `json` — 稳定 schema(`summary`、`error_groups`、`timeline`、`anomalies`、`diagnosis`),供工具与 CI 消费
- `markdown` — 适合 Issue、Postmortem、PR 评论

## 开发

```bash
go build ./...
go test ./...
go vet ./...
```

## 测试

```bash
go test ./...          # 每个核心包的单元测试
go test ./internal/engine/ -run TestIncidentFixtures -v   # 端到端测试
```

每个核心包都有 table-driven 单元测试;`testdata/incidents/*` 驱动端到端管线
测试(例如 `mysql-outage.log` 必须诊断为 `Database unavailable`,而不是 HTTP
5xx 症状)。

## 贡献

参见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 许可

[MIT](LICENSE)