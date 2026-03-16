# Flux-AI-Gateway — 需求规格说明书

**版本**: v1.0.0
**日期**: 2026-03-16
**作者**: Flux-AI-Gateway Team
**状态**: 正式发布

---

## 修订历史

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0.0 | 2026-03-16 | 初始版本，Phase 1 & 2 需求确定 |

---

## 1. 项目背景与目标

**Flux-AI-Gateway** 是一个高性能、自适应的 AI API 网关，旨在解决 3000 人规模企业环境下的 LLM 访问稳定性问题。

核心特点：
- **自适应调度**：在流式输出中途检测延迟并自动 Failover
- **极致可观测性**：量化用户体验分数（TTFT、ITL、Token Usage）

---

## 2. 技术栈约束

| 项目 | 选型 |
|------|------|
| 语言 | Golang 1.22+ |
| 框架 | Gin (HTTP Web Framework) |
| 核心库 | `net/http/httputil`, `golang.org/x/time/rate`, `github.com/prometheus/client_golang` |
| 存储 | Redis（分布式限流和状态共享） |
| 配置 | YAML（models.yaml 管理模型注册表） |

---

## 3. 核心功能需求

### A. 优先级限流 (Priority Rate Limiting)
- 识别 `X-User-ID` Header
- 支持 VIP 和 Normal 两个等级
- 逻辑：从 Redis 读取用户权限，VIP 拥有更高的令牌产生速率和最大并发数

### B. 赛马调度机制 (Hedged Requests)
- 网关同时向两个或更多后端发起请求
- 谁先成功输出有效 Token，谁就成为响应主路
- 立即取消其余请求（含 HTTP 连接释放），节省 Token 成本

### C. 流式中断检测与 Failover (Breakpoint Failover)
- 持续监测 Inter-Token Latency (ITL)
- 若 ITL > 2 秒，生成「续写 Prompt」（=原 Prompt + 已生成内容）
- 将续写请求发往 Fallback 模型，无缝接续原 HTTP 连接

### D. 优先级格指标 (Observability)
通过 Prometheus 暴露以下指标：
- `flux_ttft_seconds` (Histogram)：首 Token 延迟
- `flux_error_429_total` (Counter)：限流次数
- `flux_token_usage_total` (Counter)：Total Token 消耗

### E. 模型注册与仲裁 (Phase 3)
- YAML 驱动的模型配置，支持：主力模型、备选模型、过期日期
- Arbiter 按场景（X-Flux-Scenario Header）动态选择后端组合
- 启动时自动检测模型 EOL，过期模型自动标记 Inactive

---

## 4. 非功能性需求

- P99 网关延迟增加 < 50ms（不含模型自身延迟）
- 支持 3000 并发用户，依托 Redis 分布式限流
- 配置热更新（模型上下线无需重启）—— Phase 3+
