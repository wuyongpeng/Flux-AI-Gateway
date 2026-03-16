# Flux-AI-Gateway — 架构设计文档

**版本**: v1.1.0
**日期**: 2026-03-16
**作者**: Flux-AI-Gateway Team
**基于需求版本**: requirements-v1.0.0

---

## 修订历史

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0.0 | 2026-03-16 | Phase 1 & 2：核心代理、调度、限流 |
| v1.1.0 | 2026-03-16 | Phase 3：动态模型注册表与策略仲裁 |

---

## 1. 项目目录结构

```
.
├── cmd/
│   └── gateway/
│       └── main.go           # 程序入口，加载 Registry + Arbiter
├── internal/
│   ├── proxy/                # SSE 拦截、转发、TTFT 追踪
│   │   ├── proxy.go          # 主 HTTP Handler
│   │   └── interceptor.go    # StreamMonitor：TTFT、ITL 检测
│   ├── scheduler/            # 并发调度策略
│   │   └── hedged.go         # FastResponse：赛马 + Context 取消
│   ├── limiter/              # Redis 令牌桶限流器
│   ├── middleware/           # Prometheus 指标、Auth 中间件
│   ├── registry/             # 模型注册表
│   │   └── registry.go       # YAML 解析、EOL 检测
│   ├── arbiter/              # 策略仲裁引擎
│   │   └── arbiter.go        # 按场景选主备模型，执行路由
│   └── provider/             # 上游厂商适配器
│       ├── gemini.go         # Google Gemini REST API
│       └── mock.go           # 本地 Mock（测试用）
├── configs/
│   ├── models.yaml           # 模型注册主配置
│   └── prometheus.yml        # Prometheus 抓取配置
├── docs/                     # 设计文档与需求文档（本目录）
└── pkg/utils/                # 公共工具包
```

---

## 2. 核心组件架构

### A. `internal/proxy` — 代理核心

**StreamMonitor** (`interceptor.go`):
- 包装 `io.ReadCloser`，追踪每个字节流过
- 首个有效 Token → 记录 `TTFT`
- 相邻 Token 时间差 → 检测 `ITL`，触发 Failover 回调

**HandleGatewayRequest** (`proxy.go`):
1. 读取 `X-Flux-Scenario` Header，调用 `Arbiter.GetBackends()`
2. 调用 `scheduler.FastResponse()` 并选胜出流
3. 用 `StreamMonitor` 包裹，向客户端推送 SSE
4. 末尾注入 Metrics 摘要（TTFT、总耗时、Token 消耗）

### B. `internal/scheduler` — 赛马调度

**FastResponse** (`hedged.go`):
```
goroutine A: Provider1.SendRequest(ctx)  ─┐
                                           ├─ 第一个返回有效 token → 取消另一个
goroutine B: Provider2.SendRequest(ctx)  ─┘
```
- 使用 `context.WithCancel()` 管理各 goroutine 生命周期
- 赢家 ctx 从父 ctx 继承，保证流能继续读完

### C. `internal/registry` — 模型注册表

从 `configs/models.yaml` 解析：
- 模型 ID、所属厂商、EOL 日期、能力标签
- 启动时自动标注 `IsActive`，EOL  < 30 天提前告警

### D. `internal/arbiter` — 策略仲裁

| 场景 (Scenario) | 策略 (Strategy) | 行为 |
|---|---|---|
| `speed_racing` | hedged | 同时发两个，谁快谁赢 |
| `high_concurrency` | failover | 主先跑，429 / 错误才切备 |
| `long_context` | fallback | 主先跑，ITL > 2s 则后台续写 |

### E. `internal/provider` — 厂商适配

实现 `scheduler.BackendProvider` 接口：
```go
type BackendProvider interface {
    Name() string
    SendRequest(ctx context.Context, body []byte) Response
}
```
当前实现：`GeminiProvider`（真实 API）、`MockBackend`（本地模拟）

---

## 3. 请求全链路时序

```
Client
  │─ POST /v1/chat/completions ─────────────────────────────────────────────►│
  │  (X-Flux-Scenario: speed_racing)                                         │
  │                                                                           │ proxy.go
  │                                               arbiter.GetBackends()       │
  │                                               [Flash, Pro] strategy=hedged│
  │                                                                           │
  │                           ┌── goroutine A ──► GeminiFlash.SendRequest() ─►│ Gemini API
  │                           │                                               │
  │       scheduler.          │── goroutine B ──► GeminiPro.SendRequest()  ──►│ Gemini API
  │       FastResponse()──────┤                                               │
  │                           │                                               │
  │                           └── first token wins → cancel other ──────────►│
  │                                                                           │
  │◄─ SSE stream (TTFT + tokens + total_latency + token_usage) ──────────── │
```

---

## 4. 可观测性 Metrics

| 指标 | 类型 | 说明 |
|------|------|------|
| `flux_ttft_seconds` | Histogram | 首 Token 延迟分布 |
| `flux_error_429_total` | Counter | 限流触发次数 |
| `flux_token_usage_total` | Counter | Token 消耗量 |

SSE 流末尾客户端可见指标：
```
data: [Metrics] TTFT: 180µs
data: [Metrics] Total Answer Generation Latency: 446ms
data: [Metrics] Total Token Usage: 234
```
