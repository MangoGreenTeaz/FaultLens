# FaultLens V2 Implementation Task

> V1 已完成并验收:核心管线(input → parser → normalize → grouping → timeline → anomaly → diagnosis → output)、6 条诊断规则、3 种输出格式、9 个 fixtures、12 包测试全绿。
>
> V2 的目标是把 FaultLens 从"技术验证"推进到"真正可用的开源诊断工具":
>
> **更准(More accurate)· 更广(Wider coverage)· 更可配(More configurable)**

---

## 一、V1 遗留问题(驱动 V2 的理由)

| # | V1 局限 | V2 回应 |
|---|---|---|
| 1 | 规则关键词/阈值**硬编码**在代码里,用户无法定制 | 配置系统 + 自定义规则引擎 |
| 2 | 只有 6 条诊断规则,覆盖有限 | 新增 8 条规则,规则冲突消解 |
| 3 | 只有 4 种日志格式 | 新增 5 种 parser |
| 4 | 一次只能处理单文件/stdin | 多文件输入(glob / 目录 / 排除) |
| 5 | 单线程流式,100MB 目标 | 并发解析 + 500MB 目标 + benchmark |
| 6 | 只有 terminal/json/markdown 文本 | HTML 报告(离线单文件)+ 报告 diff |
| 7 | module 未发布,go install 不可用 | 发布工程(goreleaser / 版本标签) |

---

## 二、V2 严格范围

### 实现(V2 只做这些)

```text
配置系统            →  全局/项目/CLI 三级,阈值与规则可配置
自定义规则引擎      →  用户规则文件,与内置规则统一评估
新增诊断规则        →  8 条(disc full、证书过期、MQ、连接池、网络分区、CPU、慢查询、死锁)
新增格式 parser     →  5 种(Apache、Python、Syslog、Docker JSON、Kubernetes)
多文件输入          →  glob / 目录递归 / --exclude
性能与并发          →  worker pool 解析 + 内存优化 + 500MB benchmark
HTML 报告 + diff    →  离线单文件报告 + faultlens diff 命令
发布工程            →  版本注入、GitHub Release、go install 验证
```

### V2 明确禁止

```text
1. 不接数据库 / Elasticsearch / Kafka(作为日志源可以解析客户端日志,但不接入系统)
2. 不做 Web UI 服务 / 守护进程 / 实时监控(HTML 报告是静态单文件,不是服务)
3. 不做 LLM / AI API
4. 不做认证 / 账号 / 云服务
5. 不做分布式处理
6. 不实现复杂插件市场(配置文件即可扩展,不需要插件框架)
7. 不破坏 V1 的行为兼容(CLI 命令、JSON schema、默认输出)
8. 不引入重量级依赖(新增依赖必须论证)
```

---

## 三、技术栈

```text
Go 1.22+            (V1 已验证,保持;如工具链允许可评估升级)
Cobra               (已有)
Go standard library (核心逻辑)
gopkg.in/yaml.v3    (唯一新增依赖:配置文件解析,理由充分且轻量)
```

性能目标:500MB 日志在普通开发机正常处理。

---

## 四、核心设计原则

延续 V1:

```text
Local-first       一切本地完成
Explainable       任何诊断可追溯:Diagnosis → Evidence → Rule → Confidence
Evidence over guessing  证据不足输出 Insufficient evidence
Simple architecture      struct / interface / function / package
```

V2 新增:

```text
Configure over code   规则、阈值、关键词通过配置表达,而不是改代码
Progressive enhancement  默认配置开箱即用,配置只做增强,不引入必须项
Backward compatible   V1 的命令、输出、schema 保持可用
```

---

## 五、配置系统设计

### 配置文件优先级(低 → 高)

```text
1. 内置默认配置(编译期内置,保证开箱即用)
2. 项目级 .faultlens.yaml(仓库根目录)
3. 用户级 ~/.config/faultlens/config.yaml
4. --config <file> 显式指定(最高)
```

### 配置结构示例

```yaml
# .faultlens.yaml
anomaly:
  min_baseline: 5        # 异常检测基线最少桶数
  z_score: 3.0           # z-score 阈值
  min_increase: 3.0      # 最小倍率
  min_errors: 10         # 当前桶最少错误数

rules:
  database_unavailable:
    enabled: true
    strong_keywords: [mysql, postgres, database, sql, jdbc]
    weak_keywords: [connection refused, connection timeout]
    threshold: 10        # 强证据所需组数
  redis_unavailable:
    enabled: true

custom_rules:
  - id: disk_full
    root_cause: "Disk full"
    severity: critical
    keywords: [no space left on device, disk full]
    strong_weight: 0.40
    supporting_keywords: [write error, i/o error]
    supporting_weight: 0.20
    enable_downstream: true    # 结合后续 5xx 加分

output:
  format: terminal        # 默认输出格式
  max_groups: 5           # 报告展示错误组上限

parsers:
  json: true              # parser 开关(如某些格式不可靠可关闭)
```

### CLI 命令

```bash
faultlens config init          # 生成默认 .faultlens.yaml 到当前目录
faultlens config show          # 打印生效配置(含内置默认值合并)
faultlens config validate      # 校验配置文件(自定义规则语法)
```

### 测试要求

```text
- 三级优先级合并(table-driven)
- 未知配置项忽略不报错(向前兼容)
- 非法自定义规则:单独报错,不影响内置规则运行
- config init / show / validate 命令测试
```

---

## 六、自定义规则引擎

### 设计

```go
// diagnosis/rules 新增 CustomRule,复用现有 DiagnosisRule 接口:

type CustomRule struct {
    ID              string
    RootCause       string
    Severity        model.Severity
    Keywords        []string   // 强证据关键词
    StrongWeight    float64
    SupportingKw    []string   // 支撑证据关键词
    SupportingWt    float64
    EnableDownstream bool      // 是否计算后续 5xx 加分
    Recommendations []string
}

// 实现 diagnosis.DiagnosisRule 接口,与内置规则在同一个 Engine 中评估
```

### 校验规则

```text
- id 必须唯一(与内置规则冲突 → 报错)
- root_cause 非空
- 0 <= weight <= 1
- keywords 至少 1 个
- severity 必须是 low/medium/high/critical
```

### 测试

```text
- 自定义规则命中 → 正确诊断
- 自定义规则与内置规则同现 → 按 confidence 竞争
- 非法规则文件 → validate 报错,运行时跳过并警告
- 自定义规则不依赖文件名/硬编码
```

---

## 七、新增诊断规则(8 条)

| ID | Root Cause | 检测关键词 | Severity | 优先级备注 |
|---|---|---|---|---|
| `disk_full` | Disk full | no space left on device, disk full, write error | critical | 独立强信号 |
| `certificate_expired` | Certificate expired | certificate expired, x509, ssl handshake failed | high | 独立强信号 |
| `mq_unavailable` | Message queue unavailable | rabbitmq, amqp, kafka client, broker unavailable | critical | 上游优先于 5xx |
| `connection_pool_exhausted` | Connection pool exhausted | connection pool, pool exhausted, too many connections | high | 常为下游症状 |
| `network_partition` | Network partition | network unreachable, host unreachable, connection reset | critical | 上游优先于 5xx |
| `cpu_saturation` | CPU saturation | cpu usage, load average, throttled, cpu 100 | high | 时序相关 |
| `slow_query` | Slow query | slow query, query took, latency exceeded, slow sql | medium | 症状类 |
| `deadlock` | Deadlock detected | deadlock detected, lock timeout, deadlock found | high | 独立强信号 |

### 优先级链扩展

```text
V1: OOM > crash;上游(MQ/DB/Redis/网络) > HTTP 5xx
V2 新增原则:
  - disk_full / certificate_expired / deadlock:强独立信号,直接竞争
  - connection_pool_exhausted:若同时有 DB 不可用,降级(池耗尽是其症状)
  - 所有新增上游类规则:命中时对 5xx 规则施加 Contradict 惩罚(复用 ScoreContradict)
```

### 测试与 fixtures

```text
- 每条规则:强证据 / 弱证据 / 无证据(table-driven)
- 新增事故 fixtures:
  testdata/incidents/disk-full.log
  testdata/incidents/certificate-expired.log
  testdata/incidents/mq-outage.log
  testdata/incidents/connection-pool-exhausted.log
  testdata/incidents/network-partition.log
  testdata/incidents/cpu-saturation.log
  testdata/incidents/slow-query.log
  testdata/incidents/deadlock.log
- 集成测试:每个 fixture 断言期望根因
- 优先级:DB 不可用 + 连接池耗尽 → 优先 DB
```

---

## 八、新增格式 Parser(5 种)

| Parser | 示例行 | 关键提取 |
|---|---|---|
| **Apache** | `127.0.0.1 - - [31/Aug/2026:14:32:01 +0800] "GET /api HTTP/1.1" 200 123 "-" "Mozilla/5.0"` | 同 Nginx 字段(status/method/path/ua) |
| **Python** | `2026-08-31 14:32:01,123 ERROR root: database down` | 时间戳(带毫秒逗号)+ level + logger + message |
| **Syslog** | `<134>Aug 31 14:32:01 hostname app[123]: error message` | RFC 3164(PRI + 时间)|
| **Syslog-RFC5424** | `<134>1 2026-08-31T14:32:01.123Z hostname app 123 ID - error message` | RFC 5424 结构化 |
| **Docker JSON** | `{"log":"error: db down","stream":"stderr","time":"2026-08-31T14:32:01.123Z"}` | log/stream/time |
| **Kubernetes** | `2026-08-31T14:32:01.123Z stdout F error: db down` | RFC3339 + stream + flags |

### Auto 检测扩展

```text
优先级(插入合理位置,保持内容特征判断):
JSON → Docker JSON → Java → Syslog → Nginx → Apache → Python → Kubernetes → Plain
```

### 测试与 fixtures

```text
testdata/apache/access.log
testdata/python/app.log
testdata/syslog/syslog.log
testdata/docker/container.jsonl
testdata/kubernetes/pod.log
- 每个 parser:基本/边界/畸形行(table-driven)
- auto 检测集成测试
```

---

## 九、多文件输入

### CLI 设计

```bash
faultlens 'logs/**/*.log'          # glob 展开
faultlens logs/                    # 目录递归(默认 *.log,*.jsonl)
faultlens app.log other.log        # 多文件参数
faultlens logs/ --exclude '*.debug.log'
```

### 设计要点

```text
- 多文件合并为单一分析(统一 timeline/grouping/diagnosis)
- 每个事件 Source 字段记录来源文件
- glob 由 Go 标准库 filepath.Glob + 目录递归实现,不引第三方
- 无匹配文件 → 错误提示(列出尝试的模式)
- stdin 与多文件互斥(stdin 模式保持 V1 行为)
```

### 测试

```text
- glob 匹配、目录递归、--exclude
- 多文件合并统计正确(Source 区分)
- 无匹配 → 报错
- 与 V1 单文件行为兼容
```

---

## 十、性能与并发

### 目标

```text
V1: 100MB 可处理
V2: 500MB 普通开发机可处理,内存增量受控
```

### 实现

```text
1. 解析并发:worker pool,按行批次分发(保持输出顺序或无需有序聚合)
   - 使用 goroutine + channel,不引第三方并发库
2. 内存优化:
   - LogEvent 对象复用池(sync.Pool)减少 GC 压力
   - 错误事件切片按需扩容,非错误事件只统计不保留
   - Fields map 延迟分配
3. 基准测试:
   - internal/engine/bench_test.go:合成 100MB/500MB 日志
   - 记录:吞吐(MB/s)、峰值内存(B/op)、分配(alloc/op)
```

### 注意

```text
- 并发只加在"可安全并行"的阶段(解析),聚合阶段保持单线程确定性
- 不提前做分布式/流式管道
- benchmark 结果写入文档,不写夸张数字
```

---

## 十一、HTML 报告 + 报告 diff

### HTML 报告

```bash
faultlens app.log --output html -o report.html
```

```text
- 单文件离线 HTML(内嵌 CSS + 内联 SVG,无外部 CDN 依赖,离线可用)
- 内容:Summary / 时间线条形图(错误数/总览)/ Top 错误组 / 诊断(证据+建议)
- 生成方式:Go 标准库 html/template + 手动 SVG,不引图表库
- 可作为 CI artifact 或 Incident 分享
```

### 报告 diff

```bash
faultlens diff before.json after.json
```

```text
- 对比两次 JSON 输出(deploy 前后 / 事故前后)
- 输出:新增/消失/变化的错误组、时间线差异、诊断变化
- 复用现有 JSON schema,无新模型
```

### 测试

```text
- HTML:生成文件可打开、包含关键 section、离线无外部引用
- diff:相同 / 新增组 / 消失组 / 诊断变化
```

---

## 十二、发布工程

### 版本

```text
- 语义化版本 v0.2.0(V2 首个 release)
- ldflags 注入 version/commit/date(框架已有,完善 CI)
- faultlens version 输出稳定格式
```

### GitHub Actions release workflow

```text
.github/workflows/release.yml:
  - tag v* 触发
  - go build 多平台矩阵(linux/darwin/windows × amd64/arm64)
  - 生成压缩包 + checksums
  - 创建 GitHub Release(附 artifact)
```

### go install 验证

```text
- 决策:module path 是否迁移(github.com/faultlens/faultlens)
  方案 A:迁移到 faultlens org(需建立 org,go install 直接可用)
  方案 B:改 module path 为当前 repo(立即可用,后续迁移再改)
  方案 C:保持现状,README 标注源码构建(已有)
- V2 必须在三者中做出决定并落实,README 安装节同步
```

### 文档

```text
- README 更新(V2 功能、配置文档、新格式、新规则)
- CHANGELOG 增加 v0.2.0 记录
- CONTRIBUTING 更新(如何新增 parser/规则/配置项)
```

---

## 十三、目录演进(相对 V1)

```text
internal/
├── config/            # 新增:配置加载/合并/校验
├── parser/            # 新增 apache.go python.go syslog.go docker.go kubernetes.go
├── diagnosis/rules/   # 新增 8 规则 + custom.go(自定义规则)
├── engine/            # 多文件输入、并发、配置注入
├── output/            # 新增 html.go diff.go
├── report/            # 新增:diff 比较逻辑
└── (其余保持 V1)
testdata/              # 新增 apache/ python/ syslog/ docker/ kubernetes/ + 8 事故 fixtures
```

---

## 十四、阶段划分(开发顺序)

> 每完成一个 Phase:运行测试 → 检查代码 → 修复 → 总结 → 进入下一 Phase。
> 不要跳过测试。不要擅自扩大范围。

### Phase 1 — 配置系统

```text
目标:配置加载/合并/校验 + 3 个 config 命令
任务:
  - internal/config:内置默认值 + YAML 解析 + 三级优先级合并
  - 阈值注入:anomaly.Detector 从配置取值
  - CLI:config init / show / validate
测试:合并优先级、未知键容忍、非法值处理、命令测试
验收:go build/test/vet/gofmt 通过;阈值可通过配置改变检测行为
```

### Phase 2 — 自定义规则引擎

```text
目标:用户规则文件加载,与内置规则统一评估
任务:
  - custom.go:CustomRule 实现 DiagnosisRule
  - 规则加载:配置中的 custom_rules → 注册到 Engine
  - 校验:重复 id / 权重越界 / 缺关键词 → validate 报错
  - --config 指定规则文件
测试:命中/未命中/冲突/非法规则(重点)
验收:自定义规则可改变诊断结果;非法规则不影响内置规则
```

### Phase 3 — 新增诊断规则(8 条)

```text
任务:disk_full / certificate_expired / mq_unavailable / connection_pool_exhausted /
     network_partition / cpu_saturation / slow_query / deadlock
     每条:关键词 + 打分 + recommendations + 优先级处理
     新增 8 个事故 fixtures + 集成测试
测试:每条规则 table-driven(强/弱/无证据)+ 优先级竞争
验收:8 个 fixture 诊断正确;优先级链符合设计
```

### Phase 4 — 新增格式 Parser(5 种)

```text
任务:apache / python / syslog(RFC3164+5424)/ docker json / kubernetes
     auto 检测优先级扩展
     新增 5 个格式 fixtures
测试:基本/边界/畸形行 + auto 检测
验收:5 种格式正确解析;auto 检测正确路由
```

### Phase 5 — 多文件输入

```text
任务:glob / 目录递归 / --exclude / 多参数
     与 stdin 互斥处理
测试:匹配/排除/合并统计/无匹配报错
验收:多文件合并分析正确,Source 可区分
```

### Phase 6 — 性能与并发

```text
任务:worker pool 解析、sync.Pool 事件复用、字段延迟分配
     internal/engine/bench_test.go
测试:正确性回归(并发下结果与单线程一致)
验收:500MB 可处理;benchmark 数据记录
```

### Phase 7 — HTML 报告 + diff

```text
任务:html.go(html/template + 内联 SVG)、diff.go(faultlens diff)
测试:HTML 离线可打开、diff 场景
验收:HTML 报告可生成;diff 输出新增/消失/变化
```

### Phase 8 — 发布工程

```text
任务:release.yml(goreleaser 或手写多平台构建)、版本注入、go install 决策落地
测试:版本输出、artifact 校验和
验收:tag 触发 Release 含多平台二进制;go install 按决策可用
```

### Phase 9 — E2E + 文档 + 验收

```text
任务:V2 fixtures 全量 E2E、README/CHANGELOG/CONTRIBUTING 更新、配置文档
验收:全部验收标准满足(见下)
```

---

## 十五、最终验收标准

```text
Build:    go build ./...          成功
Test:     go test ./...           全部通过
Vet:      go vet ./...            通过
Format:   gofmt                   正确

CLI:      faultlens <file> / errors / timeline / incident / diff / config init
          全部可运行;V1 命令行为不变

配置:     config show 输出生效配置;自定义规则改变诊断结果
新规则:   8 个新事故 fixture 诊断正确;优先级链生效
新格式:   5 种格式 fixture 解析正确;auto 检测正确
多文件:   glob/目录输入正确,Source 区分
性能:     500MB 日志可处理,benchmark 有记录
HTML:     report.html 离线可打开,含关键 section
Diff:     diff 命令正确输出组/诊断变化
发布:     release workflow 产出多平台二进制;版本信息正确
文档:     README 新用户 5 分钟内可运行;V2 功能文档完整
```

---

## 十六、禁止事项(V2)

```text
1. 为了让 Demo 好看而硬编码结果。
2. 根据 fixture 文件名判断根因。
3. 根据单一关键词直接给高 confidence。
4. 随机生成 confidence。
5. 发现无法解析就 panic。
6. 为"架构完整"引入数据库/ES/Kafka。
7. 为"AI 项目"接入 LLM。
8. 添加与 V2 无关的大型依赖(yaml.v3 是唯一新增,且需论证)。
9. 复制其他项目代码。
10. 添加没有测试的核心逻辑。
11. 破坏 V1 兼容(命令、JSON schema、默认行为)。
12. 提前实现 V3 功能(实时监控/daemon/Web UI 服务等)。
```

特别注意:

> 配置与规则必须通过真实日志验证,不能因为"配置了关键词"就声称支持某类事故。

---

## 十七、V3 前瞻(只记录,不实现)

```text
- 实时 tail / daemon 模式(等待事件流)
- Web UI(本地静态服务)
- 规则学习/自动调参(需要大量标注数据)
- 跨主机多文件时序对齐
- 插件生态(v2 用配置文件,若仍不够再考虑)
```

---

开始实现时从 Phase 1 开始,每 Phase 完成即验证、总结、推送。