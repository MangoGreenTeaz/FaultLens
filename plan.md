# FaultLens V1 Implementation Task

你现在是这个开源项目的核心开发者，请从 0 开始实现一个 Go CLI 项目：

# FaultLens

Slogan：

> FaultLens — See beyond the error.

## 一、项目定位

FaultLens 是一个 local-first、offline-first 的日志事故诊断 CLI。

它不是简单的日志查看器，也不是日志搜索平台。

它的核心目标是：

> 从大量原始日志中识别错误模式、发现异常时间点、分析错误之间的关系，并基于透明、可解释的规则推断最可能的 Root Cause。

用户输入：

```bash
faultlens app.log
```

或者：

```bash
cat app.log | faultlens
```

FaultLens 最终应该输出：

```text
Incident detected

Root Cause:
MySQL became unavailable

Confidence:
91%

Evidence:
14:32:15  MySQL healthcheck failed
14:32:17  Connection refused
14:32:19  HTTP 500 spike
14:32:24  Request timeout increased

Recommended:
1. Check MySQL availability
2. Check connection limit
3. Check recent database restart
```

第一版的重点不是功能数量，而是：

1. 正确
2. 可解释
3. 可测试
4. 易维护
5. 易扩展
6. 真正可以作为 GitHub Open Source 项目使用

---

# 二、V1 的严格范围

第一版只实现：

```text
日志输入
    ↓
日志解析
    ↓
日志标准化
    ↓
错误归一化
    ↓
错误聚类
    ↓
时间线分析
    ↓
异常检测
    ↓
Root Cause Diagnosis
    ↓
Evidence
    ↓
Recommendations
    ↓
Terminal / JSON / Markdown Report
```

第一版明确禁止：

```text
1. 不做 Web UI
2. 不做数据库
3. 不做 Elasticsearch
4. 不做 Kafka
5. 不做 Kubernetes
6. 不做 Docker API 集成
7. 不做云服务
8. 不做用户系统
9. 不做认证
10. 不做账号体系
11. 不做 LLM
12. 不调用 OpenAI API
13. 不调用其他外部 AI API
14. 不需要网络
15. 不自动修改用户系统
16. 不执行危险命令
17. 不实现复杂插件市场
18. 不提前实现 V2/V3 功能
```

如果某个功能明显属于 V2，请不要为了“架构完整”提前实现。

---

# 三、技术栈

使用：

```text
Go 1.27
Cobra
Go standard library
```

尽量减少第三方依赖。

CLI 使用 Cobra。

核心分析逻辑尽可能使用 Go 标准库实现。

项目必须能够完全离线运行。

不要因为一个简单功能引入重量级依赖。

---

# 四、核心设计原则

## 1. Local-first

所有分析在本地完成。

用户不需要：

```text
API Key
Database
Server
Cloud
Network
```

---

## 2. Explainable

不能输出：

```text
Root Cause: MySQL
Confidence: 91%
```

却没有证据。

任何 Diagnosis 都必须能够回答：

> 为什么得出这个结论？

因此：

```text
Diagnosis
    ↓
Evidence
    ↓
Rule
    ↓
Confidence
```

必须能够追溯。

---

## 3. Evidence over guessing

如果证据不足：

```text
Insufficient evidence
```

而不是强行猜测。

例如：

```text
Possible root cause:
Database unavailable

Confidence:
42%

Evidence:
2 database-related errors

Insufficient evidence for a high-confidence diagnosis.
```

这是正确行为。

---

## 4. Simple architecture

不要过度设计。

不要为了以后可能出现的功能创建：

```text
20 层 abstraction
复杂 dependency injection
复杂 event bus
复杂 plugin framework
```

优先使用：

```text
struct
interface
function
package
```

解决实际问题。

---

# 五、项目目录

使用下面的目录结构：

```text
faultlens/
│
├── cmd/
│   └── faultlens/
│       └── main.go
│
├── internal/
│   ├── model/
│   │   ├── event.go
│   │   ├── group.go
│   │   ├── timeline.go
│   │   └── diagnosis.go
│   │
│   ├── input/
│   │   ├── file.go
│   │   └── stdin.go
│   │
│   ├── parser/
│   │   ├── parser.go
│   │   ├── auto.go
│   │   ├── plain.go
│   │   ├── json.go
│   │   ├── java.go
│   │   └── nginx.go
│   │
│   ├── normalize/
│   │   └── normalizer.go
│   │
│   ├── grouping/
│   │   └── grouper.go
│   │
│   ├── timeline/
│   │   └── analyzer.go
│   │
│   ├── anomaly/
│   │   └── detector.go
│   │
│   ├── diagnosis/
│   │   ├── engine.go
│   │   ├── rule.go
│   │   └── rules/
│   │       ├── database.go
│   │       ├── redis.go
│   │       ├── oom.go
│   │       ├── timeout.go
│   │       ├── http.go
│   │       └── crash.go
│   │
│   └── output/
│       ├── terminal.go
│       ├── json.go
│       └── markdown.go
│
├── testdata/
│   ├── plain/
│   ├── json/
│   ├── java/
│   ├── nginx/
│   └── incidents/
│       ├── mysql-outage.log
│       ├── redis-outage.log
│       ├── oom.log
│       ├── http-5xx.log
│       └── application-crash.log
│
├── .github/
│   └── workflows/
│       └── test.yml
│
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── CHANGELOG.md
├── LICENSE
├── go.mod
└── go.sum
```

---

# 六、统一数据模型

不同格式的日志最终必须转换成统一的 LogEvent。

设计类似：

```go
type LogEvent struct {
    Timestamp time.Time
    Level     LogLevel
    Service   string
    Message   string
    Source    string
    Fields    map[string]string
    Raw       string
    StackTrace string
}
```

根据实际设计可以增加必要字段，但不要随意增加大量字段。

Level 至少支持：

```text
TRACE
DEBUG
INFO
WARN
ERROR
FATAL
UNKNOWN
```

---

# 七、Input

支持：

## 文件

```bash
faultlens app.log
```

## stdin

```bash
cat app.log | faultlens
```

也支持：

```bash
docker logs some-container | faultlens
```

注意：

第一版只把 stdin 当作普通输入流处理。

不要实现 Docker API。

---

# 八、Parser

设计统一 Parser 接口。

例如：

```go
type Parser interface {
    Name() string
    CanParse(line string) bool
    Parse(line string) (*LogEvent, error)
}
```

第一版实现：

```text
PlainTextParser
JSONParser
JavaParser
NginxParser
AutoParser
```

---

# 九、Plain Text Parser

至少支持常见格式：

```text
2026-08-31 14:32:01 ERROR database connection failed
```

以及：

```text
2026-08-31T14:32:01Z ERROR database connection failed
```

需要尽可能识别：

```text
timestamp
level
message
```

无法识别时：

```text
level = UNKNOWN
timestamp = zero value
message = raw line
```

不要因为一行日志格式异常而导致整个程序失败。

---

# 十、JSON Parser

至少支持：

```json
{
  "timestamp": "2026-08-31T14:32:01Z",
  "level": "ERROR",
  "message": "database connection failed",
  "service": "api"
}
```

同时兼容常见字段名：

```text
timestamp
time
ts

level
severity

message
msg

service
service_name
```

未知字段放入：

```go
Fields map[string]string
```

JSON 解析失败时不要直接让整个文件失败。

应该记录解析问题并继续处理后续日志。

---

# 十一、Java / Spring Boot Parser

重点支持：

```text
2026-08-31 14:32:01 ERROR ...
```

以及：

```text
2026-08-31T14:32:01.123+08:00 ERROR ...
```

必须识别 Java Exception 和 Stack Trace。

例如：

```text
2026-08-31 14:32:01 ERROR Database query failed
java.sql.SQLException: Connection refused
    at com.example.UserService.query(UserService.java:182)
    at com.example.UserController.get(UserController.java:71)
    at ...
```

必须将上述内容视为：

```text
ONE LogEvent
```

而不是几十个 LogEvent。

StackTrace 应保存完整内容。

同时支持：

```text
Caused by:
Suppressed:
```

等常见 Java stack trace 结构。

---

# 十二、Nginx Parser

至少支持常见 access log：

```text
127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 500 123
```

提取：

```text
timestamp
HTTP method
path
status
response size
client IP
```

字段放入：

```go
Fields
```

例如：

```text
status = 500
method = GET
path = /api
```

同时支持常见 Nginx error log。

---

# 十三、Auto Detection

如果用户没有指定：

```bash
--format
```

则自动检测。

优先级：

```text
JSON
 ↓
Java
 ↓
Nginx
 ↓
Plain Text
```

不要只通过文件扩展名判断。

应该通过内容特征判断。

---

# 十四、Error Normalization

这是核心功能之一。

动态数据必须归一化。

至少支持：

```text
IPv4
IPv6
Port
UUID
数字
时间戳
URL
文件路径
Hex ID
```

例如：

输入：

```text
Connection refused 10.0.0.1:3306
Connection refused 10.0.0.2:3306
Connection refused 10.0.0.3:3306
```

归一化：

```text
Connection refused <IP>:<PORT>
```

例如：

```text
User 12345 login failed
User 83921 login failed
```

归一化：

```text
User <NUMBER> login failed
```

例如：

```text
request 550e8400-e29b-41d4-a716-446655440000 failed
```

归一化：

```text
request <UUID> failed
```

Normalizer 必须尽量避免过度归一化。

例如：

```text
HTTP 500
HTTP 404
```

不能都变成：

```text
HTTP <NUMBER>
```

因为 status code 本身具有诊断意义。

需要设计合理的 normalization 顺序和规则。

---

# 十五、Fingerprint

对 normalized message 生成稳定 fingerprint。

例如：

```text
SHA-256(normalized message)
```

Fingerprint 只用于稳定标识错误模式。

不要依赖 map iteration 或其他不稳定数据。

---

# 十六、Error Grouping

设计：

```go
type ErrorGroup struct {
    Fingerprint string
    Message     string
    Count       int
    FirstSeen   time.Time
    LastSeen    time.Time
    Examples    []LogEvent
}
```

限制 Examples 数量，例如最多保存 3~5 条。

不要为了保存 examples 无限占用内存。

最终能够输出：

```text
ERROR GROUPS

1. Connection refused <IP>:<PORT>

   Occurrences: 4381
   First seen:  14:32:17
   Last seen:   14:47:51

2. Redis timeout

   Occurrences: 1231

3. HTTP 500

   Occurrences: 982
```

默认按照：

```text
Count descending
```

排序。

---

# 十七、Timeline

默认使用：

```text
1 minute bucket
```

设计：

```go
type TimeBucket struct {
    Start    time.Time
    Total    int
    Errors   int
    Warnings int
    Fatal    int
}
```

统计每个 bucket。

例如：

```text
14:20  total=100 errors=2
14:21  total=103 errors=3
14:22  total=98  errors=1
14:23  total=102 errors=2
14:24  total=1800 errors=942
14:25  total=2100 errors=1213
```

---

# 十八、Anomaly Detection

第一版不要机器学习。

使用简单、透明、可测试的方法。

推荐：

```text
mean
standard deviation
z-score
```

或者合理的 baseline + threshold 方法。

要求：

1. 小样本不能轻易判定异常。
2. 全部日志量很低时不能产生大量假阳性。
3. baseline 必须可解释。
4. threshold 必须可配置或有明确默认值。
5. 测试中必须验证：

   * 正常数据
   * 明显 spike
   * 小样本
   * 全部相同
   * 只有一个 bucket

输出：

```text
ANOMALY DETECTED

Time:
14:24

Baseline error count:
12.4

Current error count:
942

Increase:
75.9x
```

---

# 十九、Diagnosis Engine

设计可扩展的规则接口。

例如：

```go
type DiagnosisRule interface {
    ID() string
    Evaluate(ctx DiagnosisContext) *Diagnosis
}
```

DiagnosisContext 应该包含：

```text
LogEvents
ErrorGroups
Timeline
Anomalies
```

不要让每一个 rule 自己重新解析原始日志。

---

# 二十、V1 Root Cause Rules

只实现以下 6 类。

## 1. Database unavailable

检测：

```text
connection refused
connection timeout
database
mysql
postgres
sql
jdbc
```

并结合：

```text
大量 database error
+
后续 HTTP 5xx
```

判断。

---

## 2. Redis unavailable

检测：

```text
redis
redis timeout
connection refused
cache unavailable
```

并结合请求失败。

---

## 3. OOM

检测：

```text
OutOfMemoryError
OOMKilled
out of memory
memory limit
Java heap space
```

优先级：

```text
CRITICAL
```

---

## 4. Connection timeout

检测：

```text
connection timeout
connect timeout
read timeout
i/o timeout
socket timeout
```

---

## 5. HTTP 5xx spike

检测：

```text
500
502
503
504
```

结合 Timeline 判断是否出现明显 spike。

注意：

HTTP 5xx 本身通常是“症状”，不是一定的 Root Cause。

因此如果同时发现：

```text
MySQL unavailable
+
HTTP 500 spike
```

应该优先：

```text
MySQL unavailable
```

而不是：

```text
HTTP 500 spike
```

这点非常重要。

---

## 6. Application crash

检测：

```text
fatal
panic
uncaught exception
process exited
application crashed
```

如果发现：

```text
OOM
```

则优先使用：

```text
OOM
```

而不是笼统的 Application crash。

---

# 二十一、Diagnosis 优先级

实现一个合理的 scoring system。

例如：

```text
Evidence
    ↓
Rule Score
    ↓
Bonus / Penalty
    ↓
Final Confidence
```

不要简单：

```text
if keyword exists:
    confidence = 0.9
```

这是不允许的。

Confidence 必须和 evidence 数量、强度以及冲突情况有关。

例如：

```text
Strong evidence:
+0.40

Supporting evidence:
+0.20

Temporal correlation:
+0.15

Downstream impact:
+0.15

Contradictory evidence:
-0.20
```

具体数值可以由你合理设计，但必须：

1. 可解释
2. 可测试
3. 可复现

最终限制：

```text
0 <= confidence <= 1
```

---

# 二十二、Evidence

设计：

```go
type Evidence struct {
    Timestamp time.Time
    Type      string
    Message   string
    Weight    float64
}
```

Evidence 类型可以包括：

```text
ERROR_PATTERN
ANOMALY
TIMELINE_CORRELATION
DOWNSTREAM_IMPACT
STACK_TRACE
```

输出：

```text
Evidence

14:32:15
ERROR_PATTERN
MySQL connection refused

14:32:19
ANOMALY
HTTP 500 increased 42x

14:32:19
TIMELINE_CORRELATION
Database errors preceded HTTP 500 spike by 4 seconds
```

---

# 二十三、Recommendations

每一个 Rule 都提供排查建议。

例如 MySQL：

```text
1. Check MySQL availability
2. Check database connection limit
3. Check recent database restart
4. Check network connectivity between application and database
```

Redis：

```text
1. Check Redis availability
2. Check Redis connection limit
3. Check network connectivity
```

OOM：

```text
1. Check container memory limit
2. Check application memory usage
3. Inspect recent traffic increase
4. Check heap configuration
```

所有 recommendation 都必须是：

```text
read-only
low-risk
```

不要生成：

```text
kill
restart
delete
flush
drop
```

等危险命令。

---

# 二十四、最终 Diagnosis

设计：

```go
type Diagnosis struct {
    RootCause    string
    Confidence   float64
    Severity     Severity
    Evidence     []Evidence
    Recommendations []string
}
```

如果没有足够证据：

```text
Root Cause:
Insufficient evidence

Confidence:
0.21
```

而不是强行给出结果。

---

# 二十五、CLI

使用 Cobra。

实现：

```bash
faultlens <file>
```

完整分析。

```bash
faultlens errors <file>
```

错误聚类。

```bash
faultlens timeline <file>
```

时间线。

```bash
faultlens incident <file>
```

事故分析。

支持：

```text
--format
--output
--from
--to
```

format：

```text
auto
plain
json
java
nginx
```

output：

```text
terminal
json
markdown
```

stdin：

```bash
cat app.log | faultlens
```

也应该可以：

```bash
cat app.log | faultlens incident
```

---

# 二十六、CLI UX

默认：

```bash
faultlens app.log
```

应该让用户得到一份清晰的报告。

示例：

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

Top Error Groups

1. Connection refused <IP>:<PORT>
   4,381 occurrences

2. Request timeout
   1,231 occurrences

3. HTTP 500
   982 occurrences

────────────────────────────────────────────

Diagnosis

ROOT CAUSE
MySQL became unavailable

Confidence
0.91

Evidence
...

Recommendations
...

────────────────────────────────────────────
```

Terminal 输出要清晰，但不要为了“炫酷”加入大量动画、ASCII 艺术或复杂 UI。

---

# 二十七、JSON Output

例如：

```bash
faultlens incident app.log --output json
```

输出稳定 JSON。

结构类似：

```json
{
  "summary": {},
  "error_groups": [],
  "timeline": [],
  "anomalies": [],
  "diagnosis": {
    "root_cause": "database_unavailable",
    "confidence": 0.91,
    "severity": "critical",
    "evidence": [],
    "recommendations": []
  }
}
```

JSON schema 必须稳定、容易被 GitHub Actions 或其他工具消费。

---

# 二十八、Markdown Output

例如：

```bash
faultlens incident app.log --output markdown
```

输出适合：

```text
GitHub Issue
Incident Report
Postmortem
PR comment
```

格式清晰。

---

# 二十九、错误处理

非常重要。

以下情况不能 panic：

```text
空文件
空 stdin
损坏的 JSON
未知日志格式
错误 timestamp
单行超长
部分日志无法解析
日志没有 timestamp
日志没有 level
```

程序应该尽可能继续处理。

最后报告：

```text
Parsing warnings:
37 lines could not be fully parsed
```

而不是直接：

```text
panic
```

---

# 三十、大文件处理

第一版必须支持较大的日志文件。

不要：

```go
os.ReadFile()
```

把整个文件一次性读入内存。

使用：

```go
bufio.Scanner
```

或者合适的 streaming reader。

需要考虑：

```text
Scanner 默认 token size
```

对于超长 stack trace / 单行日志要合理处理。

原则：

> 尽可能 streaming processing。

但是不需要为了第一版实现复杂的分布式处理。

---

# 三十一、测试数据

必须创建真实风格的测试日志。

至少：

```text
testdata/plain/basic.log
testdata/json/basic.jsonl
testdata/java/spring.log
testdata/nginx/access.log
```

事故：

```text
testdata/incidents/mysql-outage.log
testdata/incidents/redis-outage.log
testdata/incidents/oom.log
testdata/incidents/http-5xx.log
testdata/incidents/application-crash.log
```

其中 MySQL outage 必须模拟完整因果链：

```text
database failure
    ↓
connection refused
    ↓
application errors
    ↓
HTTP 500
```

Redis outage 同理。

OOM：

```text
memory pressure
    ↓
OOM
    ↓
application crash
```

HTTP 5xx：

```text
5xx spike
```

但没有明确上游 Root Cause 时：

```text
Insufficient evidence
```

---

# 三十二、测试要求

必须使用：

```text
go test ./...
```

并且：

```text
go vet ./...
```

通过。

核心模块必须有 unit tests：

```text
parser
normalizer
grouping
timeline
anomaly
diagnosis
output
```

大量使用 table-driven tests。

重点测试：

```text
正常情况
边界情况
空输入
错误输入
格式混合
时间戳异常
动态参数归一化
stack trace
异常 spike
证据不足
多个 Root Cause 同时出现
```

---

# 三十三、集成测试

至少实现几个完整流程测试：

```text
input log
    ↓
parser
    ↓
normalize
    ↓
group
    ↓
timeline
    ↓
anomaly
    ↓
diagnosis
    ↓
output
```

例如：

```text
mysql-outage.log
```

最终应该检测到：

```text
database unavailable
```

而不是：

```text
HTTP 500 spike
```

如果只有：

```text
HTTP 500
```

则不能强行判断数据库故障。

---

# 三十四、性能目标

V1 不需要追求极端 benchmark，但应该保证：

```text
100MB 日志
```

可以在普通开发机器上正常处理。

目标：

```text
streaming
low memory usage
no unnecessary copies
```

不要为了性能提前实现复杂并发。

如果后续 benchmark 发现瓶颈，再优化。

---

# 三十五、README

第一版必须创建一个专业 README。

README 首页至少包含：

```text
FaultLens
See beyond the error.

Features

Installation

Quick Start

Supported Logs

Example

How It Works

Architecture

CLI Reference

Output Formats

Development

Testing

Contributing

License
```

Quick Start：

```bash
faultlens app.log
```

README 必须展示一个完整的诊断示例。

不要写夸张的 benchmark。

不要声称：

```text
AI-powered
production-ready
99% accuracy
```

这些在 V1 没有依据。

---

# 三十六、Open Source 文件

创建：

```text
LICENSE
README.md
CONTRIBUTING.md
SECURITY.md
CHANGELOG.md
```

CONTRIBUTING 至少说明：

```text
如何 clone
如何运行测试
如何新增 parser
如何新增 diagnosis rule
如何提交 PR
```

SECURITY 不需要复杂，只需要说明：

```text
如何报告安全问题
```

---

# 三十七、GitHub Actions

创建：

```text
.github/workflows/test.yml
```

至少执行：

```text
go test ./...
go vet ./...
```

必要时执行：

```text
gofmt check
```

CI 必须在 Linux 环境运行。

---

# 三十八、模块设计要求

核心分析模块不能依赖 CLI。

例如：

```text
internal/parser
internal/normalize
internal/grouping
internal/anomaly
internal/diagnosis
```

不能直接调用 Cobra。

CLI 只是：

```text
input
    ↓
engine
    ↓
output
```

这样未来才能方便：

```text
CLI
GitHub Action
API
```

共享核心能力。

---

# 三十九、禁止事项

实现过程中禁止：

```text
1. 为了让 Demo 看起来更好而硬编码结果。
2. 根据测试文件名直接判断 Root Cause。
3. 根据某一个关键词直接给出高 confidence。
4. 随机生成 confidence。
5. 发现无法解析就 panic。
6. 为了“架构完整”提前加入数据库。
7. 为了“AI 项目”提前接入 LLM。
8. 添加与 V1 无关的大型依赖。
9. 复制其他项目代码。
10. 添加没有测试的核心逻辑。
```

特别注意：

> Diagnosis 必须来自日志证据，而不能读取 testdata 文件名来判断答案。

---

# 四十、开发顺序

不要一次生成整个项目。

严格按照下面顺序实现。

## Phase 1 — Project Bootstrap

完成：

```text
go.mod
cmd/faultlens/main.go
Cobra CLI
基本命令
README skeleton
GitHub Actions
```

然后运行：

```bash
go test ./...
go vet ./...
```

---

## Phase 2 — Model + Input

实现：

```text
LogEvent
LogLevel
Input
```

完成：

```text
file input
stdin input
```

写测试。

---

## Phase 3 — Parser

依次实现：

```text
plain
json
java
nginx
auto detection
```

完成 parser tests。

Java stack trace 必须重点测试。

---

## Phase 4 — Normalize + Grouping

实现：

```text
normalization
fingerprint
error grouping
```

完成大量 table-driven tests。

---

## Phase 5 — Timeline + Anomaly

实现：

```text
time buckets
error rate
anomaly detection
```

测试：

```text
normal
spike
small sample
constant data
```

---

## Phase 6 — Diagnosis

实现：

```text
Diagnosis Engine
DiagnosisRule
Evidence
Confidence
Recommendations
```

然后实现：

```text
database
redis
oom
timeout
http
crash
```

重点保证：

```text
evidence-based
```

---

## Phase 7 — Output + CLI Integration

实现：

```text
Terminal
JSON
Markdown
```

完成：

```bash
faultlens file
faultlens errors file
faultlens timeline file
faultlens incident file
```

---

## Phase 8 — End-to-End Tests

使用：

```text
testdata/incidents/*
```

进行完整测试。

确保：

```bash
go test ./...
go vet ./...
```

全部通过。

---

## Phase 9 — Documentation

完善：

```text
README.md
CONTRIBUTING.md
SECURITY.md
CHANGELOG.md
```

确保一个第一次看到 GitHub 项目的开发者可以在几分钟内运行：

```bash
faultlens example.log
```

---

# 四十一、最终验收标准

只有满足以下条件，才认为 V1 完成：

### Build

```bash
go build ./...
```

成功。

### Test

```bash
go test ./...
```

全部通过。

### Vet

```bash
go vet ./...
```

通过。

### Format

所有 Go 文件：

```bash
gofmt
```

格式正确。

### CLI

以下命令可运行：

```bash
faultlens app.log
faultlens errors app.log
faultlens timeline app.log
faultlens incident app.log
```

### stdin

以下命令可运行：

```bash
cat app.log | faultlens
```

### Diagnosis

MySQL outage fixture：

```text
database unavailable
```

能够被识别。

### Evidence

Diagnosis 必须展示证据。

### Insufficient Evidence

只有 HTTP 500，没有上游明确原因时：

```text
Insufficient evidence
```

而不是猜测 MySQL、Redis 或其他 Root Cause。

### Output

至少支持：

```text
terminal
json
markdown
```

### Documentation

README 可以让新用户快速运行项目。

---

# 四十二、最终要求

现在开始实现。

不要一次性生成所有代码。

先完成 Phase 1：

1. 检查当前工作目录。
2. 初始化 Go 项目。
3. 创建基础目录结构。
4. 创建 Cobra CLI。
5. 实现最基础的命令：

   ```bash
   faultlens --help
   faultlens version
   ```
6. 创建 README skeleton。
7. 创建 GitHub Actions。
8. 创建基础测试。
9. 运行：

   ```bash
   go test ./...
   go vet ./...
   gofmt
   ```
10. 确认 Phase 1 正常后，再继续 Phase 2。

每完成一个 Phase：

1. 运行测试。
2. 检查代码。
3. 修复问题。
4. 简要总结完成内容。
5. 再进入下一 Phase。

不要跳过测试。

不要擅自扩大项目范围。

项目最终名称统一使用：

```text
FaultLens
```

CLI 命令统一使用：

```text
faultlens
```

不要再使用：

```text
LogSleuth
LogLens
```

作为项目名、CLI 名或 README 中的产品名称。
