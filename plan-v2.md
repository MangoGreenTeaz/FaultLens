# FaultLens V2 Implementation Task

> V1 已完成并验收：核心管线
>
> `input → parser → normalize → grouping → timeline → anomaly → diagnosis → output`
>
> 已实现 6 条诊断规则、3 种输出格式、9 个 fixtures、12 包测试全部通过。
>
> V2 的目标是把 FaultLens 从“技术验证”推进到“真正可用、可集成、可贡献的开源诊断工具”。
>
> **More Accurate · More Coverage · More Configurable · More Integratable**
>
> V2 不追求堆叠复杂技术，而是围绕真实开发者使用场景完善 CLI、规则系统、日志覆盖、CI/CD 集成、报告、发布和 OSS 社区基础设施。

---

# 一、V2 核心目标

V2 聚焦四个方向：

```text
1. More Accurate
   更准确的故障诊断
   Evidence → Rule → Confidence → Diagnosis

2. More Coverage
   支持更多日志格式、更多事故类型、多文件分析

3. More Configurable
   用户可以通过配置和自定义规则扩展 FaultLens
   不需要修改源码

4. More Integratable
   可以直接进入开发者和 CI/CD 工作流
   CLI + JSON + HTML + GitHub Actions
```

最终用户体验：

```text
日志
 ↓
FaultLens
 ↓
解析
 ↓
归一化
 ↓
错误聚类
 ↓
时间线
 ↓
异常检测
 ↓
Evidence-based Diagnosis
 ↓
Terminal / JSON / HTML
 ↓
GitHub Actions / CI Artifact / PR Report
```

---

# 二、V1 遗留问题

| #  | V1 局限                | V2 回应                                                  |
| -- | -------------------- | ------------------------------------------------------ |
| 1  | 规则关键词和阈值硬编码          | 配置系统 + 自定义规则                                           |
| 2  | 只有 6 条诊断规则           | 新增 8 条规则 + 规则竞争                                        |
| 3  | 日志格式覆盖有限             | 新增 Apache / Python / Syslog / Docker JSON / Kubernetes |
| 4  | 一次只能处理单文件/stdin      | 多文件、glob、目录递归、exclude                                  |
| 5  | 大文件处理能力有限            | 500MB benchmark + 必要时并发优化                              |
| 6  | 输出主要面向终端             | HTML 离线报告                                              |
| 7  | 缺少报告对比能力             | `faultlens diff`                                       |
| 8  | 无 CI/CD 原生集成         | GitHub Actions                                         |
| 9  | 缺少 OSS 贡献基础设施        | Issue Template / PR Template / Good First Issue        |
| 10 | module / release 不完善 | GitHub Release + 多平台二进制 + go install                   |
| 11 | 新用户需要自己理解项目          | Quick Start + Demo + examples                          |

---

# 三、V2 严格范围

## 3.1 必须实现

```text
配置系统
    ↓
全局 / 项目 / CLI 三级配置
阈值、关键词、规则可配置

自定义规则引擎
    ↓
用户可以通过 YAML 定义诊断规则

新增诊断规则
    ↓
8 条

新增 Parser
    ↓
Apache
Python
Syslog RFC3164
Syslog RFC5424
Docker JSON
Kubernetes

多文件输入
    ↓
glob
目录递归
多个文件参数
exclude

GitHub Actions
    ↓
CI 日志分析
JSON artifact
HTML artifact
可选 PR/CI summary 输出

HTML Report
    ↓
完全离线
单文件
无 CDN
适合 CI Artifact 和 Incident 分享

Diff
    ↓
比较两次 JSON 分析结果

Release Engineering
    ↓
版本注入
GitHub Release
多平台构建
checksums
go install

OSS Community
    ↓
Issue Template
PR Template
CONTRIBUTING
Good First Issues
examples
Demo
```

---

# 四、明确禁止

```text
1. 不接数据库 / Elasticsearch / Kafka
2. 不做 Web UI 服务
3. 不做 daemon
4. 不做实时监控
5. 不做实时 tail
6. 不做 LLM / AI API
7. 不做认证 / 账号 / 云服务
8. 不做分布式处理
9. 不做复杂插件市场
10. 不引入重量级依赖
11. 不破坏 V1 CLI 行为
12. 不破坏 V1 JSON schema
13. 不为了 Demo 硬编码诊断结果
14. 不根据 fixture 文件名判断根因
15. 不根据单一关键词直接给高 confidence
16. 不随机生成 confidence
17. 不复制其他项目代码
18. 不添加没有测试的核心逻辑
19. 不为了“AI 项目”而接入 AI
20. 不实现 V3 功能
```

---

# 五、技术栈

```text
Go 1.22+

Cobra
    CLI

Go standard library
    核心逻辑
    parser
    template
    filepath
    concurrency
    testing

gopkg.in/yaml.v3
    唯一新增第三方依赖
    仅用于 YAML 配置解析
```

原则：

```text
优先标准库
最少依赖
简单架构
容易贡献
容易测试
容易维护
```

任何新增依赖必须说明：

```text
1. 为什么标准库不能解决
2. 为什么该依赖值得引入
3. 对二进制体积和维护成本的影响
```

如果没有充分理由，不允许引入。

---

# 六、核心设计原则

## 6.1 Local-first

所有日志分析默认在本地完成。

```text
Local File
    ↓
FaultLens
    ↓
Local Result
```

不依赖云服务。

---

## 6.2 Explainable

任何诊断必须能够追溯：

```text
Diagnosis
    ↓
Confidence
    ↓
Rule
    ↓
Evidence
```

不能只输出：

```text
Root Cause: Database unavailable
```

必须尽可能说明：

```text
Root Cause: Database unavailable

Confidence: 0.92

Evidence:
- connection refused
- database timeout
- jdbc connection failure

Rule:
database_unavailable
```

---

## 6.3 Evidence over guessing

如果证据不足：

```text
Insufficient evidence
```

而不是强行给出根因。

---

## 6.4 Configure over code

用户应该优先通过配置：

```text
threshold
keyword
severity
weight
recommendation
```

改变行为。

不应该为了修改诊断规则而修改 Go 源码。

---

## 6.5 Progressive enhancement

默认：

```text
faultlens app.log
```

必须开箱即用。

高级功能：

```text
.faultlens.yaml
custom_rules
GitHub Actions
HTML
diff
```

都是增强能力，不应该成为最基本使用的强制依赖。

---

## 6.6 Backward compatible

V1：

```text
CLI
JSON schema
默认输出
```

必须保持兼容。

如果确实需要破坏兼容，必须：

```text
1. 明确说明
2. 更新 CHANGELOG
3. 提供迁移说明
```

默认禁止破坏兼容。

---

## 6.7 Contribution-friendly

V2 的代码结构必须让外部贡献者容易：

```text
新增 Parser
新增 Diagnosis Rule
新增 Fixture
新增 Test
修改 Documentation
```

不允许为了抽象而抽象。

---

# 七、配置系统

## 7.1 配置优先级

从低到高：

```text
1. Built-in defaults

2. Project config
   .faultlens.yaml

3. User config
   ~/.config/faultlens/config.yaml

4. Explicit config
   --config <file>
```

高优先级覆盖低优先级。

---

## 7.2 示例

```yaml
anomaly:
  min_baseline: 5
  z_score: 3.0
  min_increase: 3.0
  min_errors: 10

rules:
  database_unavailable:
    enabled: true
    strong_keywords:
      - mysql
      - postgres
      - database
      - sql
      - jdbc
    weak_keywords:
      - connection refused
      - connection timeout
    threshold: 10

  redis_unavailable:
    enabled: true

custom_rules:
  - id: disk_full
    root_cause: "Disk full"
    severity: critical
    keywords:
      - no space left on device
      - disk full
    strong_weight: 0.40
    supporting_keywords:
      - write error
      - i/o error
    supporting_weight: 0.20
    enable_downstream: true
    recommendations:
      - "Check filesystem capacity"
      - "Clean temporary files"

output:
  format: terminal
  max_groups: 5

parsers:
  json: true
```

---

# 八、Config CLI

必须支持：

```bash
faultlens config init
faultlens config show
faultlens config validate
```

## config init

生成：

```text
.faultlens.yaml
```

提供完整、合理、可读的默认配置。

---

## config show

打印最终生效配置。

必须体现：

```text
Built-in
+
Project
+
User
+
CLI
```

合并后的结果。

---

## config validate

检查：

```text
YAML syntax
unknown / invalid values
rule id
weight
severity
keywords
```

非法配置必须明确报错。

---

# 九、自定义规则引擎

自定义规则必须复用现有：

```go
DiagnosisRule
```

接口。

建议结构：

```go
type CustomRule struct {
    ID               string
    RootCause        string
    Severity         model.Severity
    Keywords         []string
    StrongWeight     float64
    SupportingKw     []string
    SupportingWt     float64
    EnableDownstream bool
    Recommendations  []string
}
```

---

## 规则校验

必须满足：

```text
id 唯一

root_cause 非空

0 <= weight <= 1

keywords 至少 1 个

severity:
low
medium
high
critical
```

与内置规则 ID 冲突：

```text
直接报错
```

---

## 规则竞争

多个规则同时命中：

```text
统一进入 Diagnosis Engine
        ↓
Evidence scoring
        ↓
Confidence
        ↓
Priority / Contradict
        ↓
最终诊断
```

不能：

```text
if keyword == xxx:
    return root cause
```

---

# 十、新增诊断规则

新增：

```text
disk_full
certificate_expired
mq_unavailable
connection_pool_exhausted
network_partition
cpu_saturation
slow_query
deadlock
```

---

## 规则设计

### disk_full

```text
no space left on device
disk full
write error
```

Severity:

```text
critical
```

---

### certificate_expired

```text
certificate expired
x509
ssl handshake failed
```

Severity:

```text
high
```

---

### mq_unavailable

```text
rabbitmq
amqp
kafka client
broker unavailable
```

Severity:

```text
critical
```

---

### connection_pool_exhausted

```text
connection pool
pool exhausted
too many connections
```

Severity:

```text
high
```

如果同时存在：

```text
database unavailable
```

应根据现有优先级逻辑考虑：

```text
DB unavailable
>
connection pool exhausted
```

---

### network_partition

```text
network unreachable
host unreachable
connection reset
```

Severity：

```text
critical
```

---

### cpu_saturation

```text
cpu usage
load average
throttled
cpu 100
```

Severity：

```text
high
```

必须考虑时间序列信息，不能仅依赖关键词。

---

### slow_query

```text
slow query
query took
latency exceeded
slow sql
```

Severity：

```text
medium
```

属于症状型诊断。

---

### deadlock

```text
deadlock detected
lock timeout
deadlock found
```

Severity：

```text
high
```

---

# 十一、诊断优先级

延续 V1：

```text
OOM
>
Crash

上游依赖
>
HTTP 5xx
```

V2 扩展：

```text
独立强信号
    disk_full
    certificate_expired
    deadlock

上游故障
    database
    redis
    mq
    network

资源问题
    connection_pool
    cpu

症状
    slow_query
    HTTP 5xx
```

但不要简单硬编码绝对优先级。

最终应该综合：

```text
Evidence
Confidence
Severity
Contradict
Temporal relation
```

进行竞争。

---

# 十二、新增 Parser

V2 新增：

```text
Apache
Python
Syslog RFC3164
Syslog RFC5424
Docker JSON
Kubernetes
```

注意：

```text
Syslog RFC3164
Syslog RFC5424
```

属于两个具体格式支持，但可以共用 Syslog parser 模块。

---

## Apache

示例：

```text
127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 200 123 "-" "Mozilla/5.0"
```

提取：

```text
timestamp
method
path
status
user agent
```

---

## Python

示例：

```text
2026-08-31 14:32:01,123 ERROR root: database down
```

提取：

```text
timestamp
level
logger
message
```

---

## Syslog RFC3164

示例：

```text
<134>Aug 31 14:32:01 hostname app[123]: error message
```

提取：

```text
PRI
timestamp
hostname
process
pid
message
```

---

## Syslog RFC5424

示例：

```text
<134>1 2026-08-31T14:32:01.123Z hostname app 123 ID - error message
```

提取：

```text
PRI
version
timestamp
hostname
app
pid
message
```

---

## Docker JSON

示例：

```json
{
  "log": "error: db down",
  "stream": "stderr",
  "time": "2026-08-31T14:32:01.123Z"
}
```

提取：

```text
log
stream
time
```

---

## Kubernetes

示例：

```text
2026-08-31T14:32:01.123Z stdout F error: db down
```

提取：

```text
timestamp
stream
flags
message
```

---

# 十三、Auto Detection

扩展 parser 检测：

```text
JSON
↓
Docker JSON
↓
Java
↓
Syslog
↓
Nginx
↓
Apache
↓
Python
↓
Kubernetes
↓
Plain
```

必须避免：

```text
一个 parser 错误匹配后阻止后续 parser
```

如果格式不确定：

```text
fallback → Plain
```

不能 panic。

---

# 十四、多文件输入

支持：

```bash
faultlens 'logs/**/*.log'

faultlens logs/

faultlens app.log other.log

faultlens logs/ --exclude '*.debug.log'
```

---

## 设计

多个文件：

```text
File A
File B
File C
   ↓
统一分析
   ↓
timeline
grouping
anomaly
diagnosis
```

每个事件必须保留：

```text
Source
```

用于追踪：

```text
error originated from which file
```

---

## stdin

保持 V1：

```text
stdin
```

与多文件输入互斥。

---

# 十五、GitHub Actions / CI Integration

这是 V2 的重点新增能力。

目标：

> 让 FaultLens 可以直接进入开发者 CI/CD 工作流。

---

## 15.1 第一阶段：CLI CI Friendly

确保：

```bash
faultlens app.log --output json
```

可以稳定用于：

```text
GitHub Actions
GitLab CI
Jenkins
其他 CI
```

要求：

```text
stdout
stderr
exit code
JSON schema
```

行为稳定。

---

## 15.2 GitHub Actions 示例

README 必须提供类似：

```yaml
- name: Analyze logs with FaultLens
  run: |
    faultlens ./logs/*.log --output json > faultlens.json
```

然后：

```yaml
- name: Generate HTML report
  run: |
    faultlens ./logs/*.log --output html -o faultlens-report.html
```

---

## 15.3 官方 Action

V2 可以提供：

```text
.github/actions/
```

或者独立 action 目录。

目标使用体验：

```yaml
- uses: MangoGreenTeaz/FaultLens@v0
  with:
    logs: ./logs
    output: html
```

如果实现 GitHub Action 需要引入额外复杂依赖：

```text
优先保持简单
优先复用 faultlens CLI
不要实现复杂 GitHub App
```

---

## 15.4 CI Artifact

GitHub Actions 中应该可以产出：

```text
faultlens.json
faultlens-report.html
```

用途：

```text
Debug failed CI
Incident investigation
Post-deployment analysis
```

---

## 15.5 CI Summary

如果成本可控，增加：

```text
GitHub Actions Job Summary
```

例如：

```text
FaultLens Analysis

Root Cause:
Database unavailable

Confidence:
0.92

Evidence:
3 strong signals
17 supporting signals

Affected period:
14:32:01 - 14:32:17
```

要求：

```text
不能依赖 GitHub API
不能要求账号
不能引入服务端
```

使用 CI 提供的环境能力即可。

---

## 15.6 PR Comment

V2 不强制实现 GitHub App。

如果能够低复杂度实现：

```text
PR / Workflow Summary
```

可以支持。

但禁止为了 PR Comment 引入：

```text
OAuth
database
server
GitHub App backend
```

V2 的原则：

```text
CLI first
CI friendly
No server
```

---

# 十六、HTML Report

命令：

```bash
faultlens app.log --output html -o report.html
```

要求：

```text
单 HTML 文件
完全离线
无 CDN
无外部资源
内嵌 CSS
内联 SVG
```

---

## 内容

```text
Summary

Timeline

Error Groups

Top Errors

Diagnosis

Confidence

Evidence

Recommendations

Source Files
```

---

## 设计目标

HTML Report 必须适合：

```text
CI Artifact
Incident sharing
Offline debugging
Code review
Postmortem
```

不能变成 Web UI 服务。

---

# 十七、Diff

命令：

```bash
faultlens diff before.json after.json
```

比较：

```text
Error Groups
Timeline
Diagnosis
Confidence
```

输出：

```text
Added

Removed

Changed
```

示例：

```text
+ database unavailable
- redis timeout

Diagnosis changed:
Redis unavailable
→
Database unavailable

Confidence:
0.71
→
0.93
```

复用已有 JSON schema。

不引入新的持久化格式。

---

# 十八、性能与 Benchmark

V2 目标：

```text
500MB 普通开发机可处理
```

但：

> 性能优化以 benchmark 为依据，不为了“架构完整”提前复杂化。

---

## 第一阶段

先实现：

```text
正确性
+
500MB benchmark
+
内存统计
```

记录：

```text
Throughput MB/s

Peak memory

B/op

alloc/op
```

---

## 第二阶段

只有 benchmark 证明存在明显瓶颈时，才考虑：

```text
worker pool
sync.Pool
对象复用
字段延迟分配
```

原则：

```text
Correctness first
Benchmark second
Optimization third
```

并发只用于：

```text
可以安全并行的 parser / processing stage
```

聚合与诊断必须保持确定性。

---

# 十九、Release Engineering

版本：

```text
v0.2.0
```

版本信息：

```text
version
commit
build date
```

使用：

```text
ldflags
```

---

## GitHub Release

tag：

```text
v*
```

触发：

```text
Linux
macOS
Windows

amd64
arm64
```

生成：

```text
binary
archive
checksums
```

---

## go install

V2 必须完成 module path 决策。

优先：

```text
github.com/MangoGreenTeaz/FaultLens
```

如果当前仓库名和 Go module path 不一致：

```text
统一 module path
```

目标：

```bash
go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest
```

必须在真实环境验证。

不要为了未来可能迁移而使用不存在的：

```text
github.com/faultlens/faultlens
```

除非该组织和仓库实际存在。

---

# 二十、OSS Community Infrastructure

这是 V2 新增重点。

---

## 20.1 CONTRIBUTING

必须说明：

```text
如何运行项目

如何运行测试

如何新增 Parser

如何新增 Diagnosis Rule

如何新增 Fixture

如何新增配置项

代码规范

提交 PR 流程
```

---

## 20.2 Issue Template

增加：

```text
Bug Report

Feature Request
```

Bug Report 至少包含：

```text
FaultLens version

OS

Go version

Input format

Expected behavior

Actual behavior

Minimal log example
```

---

## 20.3 Pull Request Template

要求贡献者说明：

```text
What changed?

Why?

How tested?

Breaking change?

Related issue?
```

---

## 20.4 Good First Issues

至少准备一些真正适合外部贡献者的任务：

```text
good first issue:
Add parser fixture

good first issue:
Improve documentation

good first issue:
Add diagnosis rule test

good first issue:
Add malformed log test

good first issue:
Improve CLI error message
```

不要创建虚假的 Issue。

---

# 二十一、Contribution-friendly Architecture

目录结构：

```text
internal/
├── config/
├── parser/
│   ├── apache.go
│   ├── python.go
│   ├── syslog.go
│   ├── docker.go
│   └── kubernetes.go
├── diagnosis/
│   └── rules/
├── engine/
├── output/
├── report/
└── ...
```

---

## 新增 Parser 流程

README 必须说明：

```text
1. Create parser
2. Implement Parser interface
3. Add fixture
4. Add table-driven tests
5. Register parser
6. Update documentation
7. Submit PR
```

---

## 新增 Rule 流程

```text
1. Create rule
2. Define evidence
3. Define scoring
4. Add fixture
5. Add tests
6. Add recommendation
7. Update documentation
8. Submit PR
```

目标：

> 一个熟悉 Go 的贡献者应该能够在较短时间内理解如何新增一个 Parser 或 Diagnosis Rule。

---

# 二十二、Examples / Demo

新增：

```text
examples/
├── basic/
├── database-outage/
├── disk-full/
├── network-partition/
└── ci/
```

每个 example：

```text
input log
expected diagnosis
README
```

---

## Quick Start

README 必须让新用户在 5 分钟内完成：

```text
Install
↓
Run
↓
Analyze
↓
Understand result
↓
Generate report
```

例如：

```bash
go install github.com/MangoGreenTeaz/FaultLens/cmd/faultlens@latest

faultlens examples/database-outage/app.log
```

---

## Demo

README 顶部应该提供：

```text
Demo GIF / Screenshot
```

展示：

```text
log
 ↓
faultlens
 ↓
diagnosis
 ↓
evidence
```

不要使用假的输出。

Demo 必须来自真实执行结果。

---

# 二十三、README 重构

README 应至少包含：

```text
1. Project description

2. Why FaultLens

3. Features

4. Demo

5. Quick Start

6. Installation

7. Supported Log Formats

8. Diagnosis Rules

9. Configuration

10. Custom Rules

11. GitHub Actions

12. HTML Report

13. Diff

14. Architecture

15. Examples

16. Contributing

17. Security

18. Roadmap

19. License
```

README 的第一屏必须让用户理解：

```text
FaultLens 是什么
解决什么问题
为什么有用
30 秒如何运行
```

---

# 二十四、测试策略

所有核心功能必须测试。

要求：

```text
table-driven tests
fixtures
integration tests
E2E tests
```

---

## Config

测试：

```text
默认配置

Project override

User override

CLI override

非法 YAML

非法 rule

unknown key

config init

config show

config validate
```

---

## Custom Rules

测试：

```text
命中

不命中

强证据

弱证据

规则竞争

ID 冲突

weight 越界

缺失 keyword
```

---

## Diagnosis Rules

每条新增规则：

```text
Strong evidence
Weak evidence
No evidence
```

并提供事故 fixture。

---

## Parser

每个 parser：

```text
Normal

Boundary

Malformed

Auto detection
```

---

## Multi-file

测试：

```text
glob

directory

recursive

exclude

multiple files

source field

no match

stdin compatibility
```

---

## HTML

测试：

```text
file generated

valid HTML

key sections exist

no external CDN

offline
```

---

## Diff

测试：

```text
same

added

removed

changed

diagnosis changed
```

---

## GitHub Actions

至少测试：

```text
CLI can run in CI

JSON output is stable

HTML artifact can be generated

exit code is deterministic
```

---

# 二十五、阶段划分

每个 Phase：

```text
实现
↓
测试
↓
go build
↓
go test
↓
go vet
↓
gofmt
↓
检查代码
↓
修复
↓
总结
↓
进入下一阶段
```

禁止跳过测试。

禁止擅自扩大范围。

---

# Phase 1 — Configuration

目标：

```text
配置加载
配置合并
配置校验
```

实现：

```text
internal/config

Built-in defaults

Project config

User config

--config

config init

config show

config validate
```

验收：

```text
go build ./...
go test ./...
go vet ./...
gofmt
```

并验证：

```text
配置阈值真的可以改变分析行为
```

---

# Phase 2 — Custom Rule Engine

实现：

```text
CustomRule

rule loader

validation

engine integration
```

验收：

```text
自定义规则可以真正改变诊断结果

非法规则不会破坏内置规则

规则竞争结果确定

测试完整
```

---

# Phase 3 — GitHub Actions / CI Integration

这是 V2 的核心阶段。

实现：

```text
CLI CI-friendly behavior

JSON output

HTML artifact

GitHub Actions example

GitHub Actions workflow

Job Summary（低复杂度实现时）

官方 Action（如果实现成本可控）
```

禁止：

```text
GitHub App backend

OAuth

database

server
```

验收：

```text
真实 GitHub Actions workflow 成功运行 FaultLens

可以产出 JSON artifact

可以产出 HTML artifact

README 有完整使用示例
```

---

# Phase 4 — New Diagnosis Rules

新增：

```text
8 rules
```

每条：

```text
rule

evidence

scoring

recommendation

fixture

tests
```

验收：

```text
8 个事故 fixture 正确诊断
```

---

# Phase 5 — New Parsers

新增：

```text
Apache

Python

Syslog

Docker JSON

Kubernetes
```

验收：

```text
正常日志

边界日志

畸形日志

auto detection
```

全部测试。

---

# Phase 6 — Multi-file Input

实现：

```text
glob

directory

recursive

exclude

multiple arguments

Source
```

验收：

```text
多文件统一分析

timeline 正确

grouping 正确

diagnosis 正确

Source 正确
```

---

# Phase 7 — HTML Report + Diff

实现：

```text
HTML

SVG timeline

Diagnosis section

Evidence section

Recommendations

faultlens diff
```

验收：

```text
HTML 完全离线

没有外部 CDN

diff 正确识别：

added
removed
changed
```

---

# Phase 8 — Performance + Benchmark

第一阶段：

```text
500MB benchmark

memory measurement

alloc measurement
```

然后根据 benchmark 决定是否需要：

```text
worker pool

sync.Pool

对象复用
```

验收：

```text
500MB 可正常处理

benchmark 有真实数据

结果可重复

并发结果与单线程结果一致
```

不要为了满足“并发”而强行增加复杂度。

---

# Phase 9 — Release Engineering

实现：

```text
version

ldflags

release workflow

multi-platform build

checksum

GitHub Release

go install
```

验收：

```text
tag v0.2.0
↓
GitHub Release
↓
多平台 artifact
↓
checksum
↓
go install
↓
faultlens version
```

---

# Phase 10 — OSS Community + Documentation

实现：

```text
README

CONTRIBUTING

CHANGELOG

SECURITY

Issue Templates

PR Template

Good First Issues

examples

Demo
```

验收：

```text
新用户 5 分钟可以运行

贡献者可以根据 CONTRIBUTING 新增 Parser

贡献者可以根据 CONTRIBUTING 新增 Rule

README 包含 GitHub Actions 示例

README 包含 Demo

README 包含配置文档
```

---

# Phase 11 — Final E2E + Release Candidate

执行完整检查：

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

然后：

```text
所有 fixtures

所有 parser

所有 rules

config

custom rules

multi-file

HTML

diff

GitHub Actions

release

go install
```

全部进行 E2E 验证。

最后：

```text
创建 v0.2.0 Release Candidate
```

---

# 二十六、最终验收标准

## Build

```text
go build ./...
```

必须成功。

---

## Test

```text
go test ./...
```

全部通过。

---

## Vet

```text
go vet ./...
```

通过。

---

## Format

```text
gofmt -l .
```

不得出现需要格式化的 Go 文件。

---

## CLI

以下命令必须正常：

```text
faultlens <file>

faultlens errors

faultlens timeline

faultlens incident

faultlens diff

faultlens config init

faultlens config show

faultlens config validate
```

V1 命令行为保持兼容。

---

## Configuration

必须满足：

```text
默认配置可运行

项目配置生效

用户配置生效

CLI config 生效

config show 正确

config validate 正确
```

---

## Custom Rule

必须满足：

```text
用户规则可以改变诊断结果

非法规则正确报错

内置规则不受非法规则影响
```

---

## Diagnosis

```text
8 条新增规则全部有 fixture

全部有单元测试

全部有集成测试

优先级竞争正确
```

---

## Parser

```text
Apache

Python

Syslog RFC3164

Syslog RFC5424

Docker JSON

Kubernetes
```

全部有：

```text
normal

boundary

malformed

auto detection
```

测试。

---

## Multi-file

```text
glob

directory

recursive

exclude

multiple files
```

全部正确。

---

## Performance

```text
500MB 日志可处理

benchmark 有记录

无明显异常内存增长
```

---

## HTML

```text
单文件

离线

无 CDN

包含 Summary

包含 Timeline

包含 Error Groups

包含 Diagnosis

包含 Evidence

包含 Recommendations
```

---

## Diff

必须正确输出：

```text
新增错误组

消失错误组

变化错误组

诊断变化

confidence 变化
```

---

## GitHub Actions

必须至少支持：

```text
FaultLens CLI

JSON output

HTML artifact
```

并提供真实可运行的 workflow 示例。

---

## Release

必须：

```text
GitHub Release

多平台 binary

checksums

version

go install
```

全部可验证。

---

## OSS

必须：

```text
README

CONTRIBUTING

CHANGELOG

SECURITY

Issue Template

PR Template

Good First Issues

Examples
```

---

# 二十七、V2 成功指标

V2 不仅以代码数量作为成功标准。

重点指标：

```text
1. 新用户可以在 5 分钟内运行

2. CLI 可以进入 CI/CD

3. GitHub Actions 可以直接分析日志

4. JSON / HTML 可以作为 CI Artifact

5. 用户可以通过 YAML 自定义规则

6. 新 Parser / Rule 容易贡献

7. 项目有正式 Release

8. README 有真实 Demo

9. 项目持续产生真实 Commit

10. Issues / PR 可以自然产生
```

不要人为制造：

```text
fake stars
fake downloads
fake issues
fake PR
```

社区指标必须来自真实用户。

---

# 二十八、OSS 项目维护策略

V2 完成后不要立即停止开发。

维护策略：

```text
Bug
    ↓
Issue
    ↓
Fix
    ↓
Test
    ↓
Release
```

Feature：

```text
Issue
    ↓
Discussion
    ↓
Implementation
    ↓
PR
    ↓
Review
    ↓
Merge
```

重点关注：

```text
真实用户反馈

Parser requests

Diagnosis rule requests

CI integration feedback

Performance issues
```

---

# 二十九、Codex for OSS 导向原则

FaultLens V2 不应该为了申请 Codex for OSS 而制造功能。

真正应该证明：

```text
FaultLens
    ↓
真实 OSS 项目
    ↓
真实开发者问题
    ↓
真实开源使用场景
    ↓
持续维护
```

尤其强调：

```text
Developer tooling

CI/CD integration

Explainable diagnosis

Local-first

Open source

Contribution-friendly
```

Codex 如果参与开发，应优先帮助：

```text
Implement features

Write tests

Review code

Fix bugs

Improve documentation

Maintain CI

Triage issues
```

但项目本身不能变成：

```text
AI demo
```

---

# 三十、V3 前瞻

V3 只记录，不实现：

```text
1. Real-time tail

2. Daemon mode

3. Web UI

4. Rule learning

5. Automatic threshold tuning

6. Cross-host timeline alignment

7. Plugin ecosystem

8. Advanced CI integrations

9. GitHub App

10. Optional LLM-assisted diagnosis
```

V2 禁止提前实现这些功能。

---

# 三十一、最终 V2 产品形态

最终：

```text
                    FaultLens
                        │
        ┌───────────────┼────────────────┐
        │               │                │
        ↓               ↓                ↓
      CLI             CI/CD          Incident
        │               │                │
        ↓               ↓                ↓
    Terminal       GitHub Actions    HTML Report
        │               │                │
        └───────────────┼────────────────┘
                        ↓
                Evidence-based
                   Diagnosis
                        │
              ┌─────────┴─────────┐
              ↓                   ↓
        Built-in Rules       Custom Rules
              │                   │
              └─────────┬─────────┘
                        ↓
                Explainable RCA
```

核心定位：

> **FaultLens is a local-first, explainable log incident diagnosis toolkit for developers and CI/CD workflows.**

---

# 三十二、执行规则

开始实现时：

```text
从 Phase 1 开始。

每完成一个 Phase：

1. 运行测试
2. 运行 go build
3. 运行 go vet
4. 运行 gofmt
5. 检查代码
6. 修复问题
7. 总结本阶段修改
8. 确认验收标准
9. 再进入下一 Phase
```

必须：

```text
不跳 Phase

不擅自扩大范围

不实现 V3

不引入未经论证的依赖

不破坏 V1

不硬编码 Demo

不根据 fixture 文件名判断结果

不随机生成 confidence

不因为“看起来更高级”而增加复杂架构
```

最终目标不是：

> **写更多代码。**

而是：

> **把 FaultLens 做成一个真实开发者可以安装、使用、集成、扩展和贡献的开源工具。**
