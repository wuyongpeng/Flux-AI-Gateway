# Flux-AI-Gateway — 多厂商模型管理设计

**版本**: v1.2.0
**日期**: 2026-03-16
**作者**: Flux-AI-Gateway Team
**状态**: 设计评审阶段
**继承自**: design-v1.1.0

---

## 修订历史

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0.0 | 2026-03-16 | Phase 1 & 2：核心代理 |
| v1.1.0 | 2026-03-16 | Phase 3：单厂商动态注册 |
| v1.2.0 | 2026-03-16 | Phase 4：多厂商统一调度 |

---

## 1. 问题陈述

当前 `v1.1.0` 架构的局限：

- `GeminiProvider` 被硬编码为唯一厂商适配器
- `configs/models.yaml` 中的模型没有厂商归属，无从路由到正确的厂商 API Key
- 无法在同一场景中跨厂商竞速（如 Gemini Flash vs OpenAI GPT-4o-mini）

---

## 2. 多厂商架构设计

### 2.1 新增目录结构

```
internal/provider/
├── gemini.go          # Google Gemini 适配器（已有）
├── openai.go          # OpenAI 适配器（新增）
├── anthropic.go       # Anthropic Claude 适配器（新增）
├── factory.go         # 厂商工厂函数：根据 provider 字段实例化
└── mock.go            # 本地测试 Mock

configs/
├── models.yaml        # 统一模型注册表（新结构）
└── prometheus.yml
```

### 2.2 新版 models.yaml 结构

模型注册表增加以下分层：
1. **`providers`**: 厂商级别，保存 API Key 引用和基础 URL
2. **`models`**: 具体模型，通过 `provider_id` 关联厂商
3. **`scenarios`**: 不变，通过模型 ID 组合路由

```yaml
providers:
  - id: "google"
    name: "Google Gemini"
    api_key_env: "FLUX_GEMINI_API_KEY"
    base_url: "https://generativelanguage.googleapis.com/v1beta"

  - id: "openai"
    name: "OpenAI"
    api_key_env: "FLUX_OPENAI_API_KEY"
    base_url: "https://api.openai.com/v1"

  - id: "anthropic"
    name: "Anthropic"
    api_key_env: "FLUX_ANTHROPIC_API_KEY"
    base_url: "https://api.anthropic.com/v1"

models:
  - id: "gemini-2.5-flash"
    provider: "google"
    eol_date: "2099-12-31"

  - id: "gpt-4o-mini"
    provider: "openai"
    eol_date: "2099-12-31"

  - id: "claude-3-5-haiku-latest"
    provider: "anthropic"
    eol_date: "2099-12-31"

scenarios:
  speed_racing:
    primary: "gemini-2.5-flash"
    backup: "gpt-4o-mini"     # 跨厂商竞速！
    strategy: "hedged"

  cost_saving:
    primary: "gemini-2.5-flash"
    backup: "claude-3-5-haiku-latest"
    strategy: "failover"
```

### 2.3 Provider Factory

Arbiter 通过 `factory.go` 的工厂函数，按厂商 ID 自动实例化不同 Provider：

```go
// internal/provider/factory.go
func NewProvider(modelID, providerID, apiKey, baseURL string) (scheduler.BackendProvider, error) {
    switch providerID {
    case "google":
        return NewGeminiProvider(modelID, apiKey), nil
    case "openai":
        return NewOpenAIProvider(modelID, apiKey, baseURL), nil
    case "anthropic":
        return NewAnthropicProvider(modelID, apiKey, baseURL), nil
    default:
        return nil, fmt.Errorf("unknown provider: %s", providerID)
    }
}
```

### 2.4 Registry 扩展

`ModelRegistry` 读取 `providers` 区块，构建 `providerMap`：

```go
type ProviderConfig struct {
    ID        string
    Name      string
    APIKeyEnv string // 从环境变量读 Key
    BaseURL   string
}

// registry.go 在初始化时读取 os.Getenv(provider.APIKeyEnv)
// 存储到 providerMap[providerID].APIKey
```

### 2.5 Arbiter 扩展

`GetBackends()` 在查找模型后，额外拿到厂商配置，然后走工厂函数：

```go
model, _ := r.GetModel(scen.Primary)
prov, _ := r.GetProvider(model.ProviderID)
backend, _ := provider.NewProvider(model.ID, prov.ID, prov.APIKey, prov.BaseURL)
```

---

## 3. 多厂商环境变量管理

`.env` 文件模板扩展：

```bash
# Google Gemini
export FLUX_GEMINI_API_KEY=your_gemini_key_here

# OpenAI
export FLUX_OPENAI_API_KEY=your_openai_key_here

# Anthropic  
export FLUX_ANTHROPIC_API_KEY=your_anthropic_key_here
```

---

## 4. 未来扩展：账户余额查询

提供可选接口，支持厂商暴露的余额/配额查询：

```go
type BalanceChecker interface {
    GetAvailableBalance(ctx context.Context) (float64, string, error)
    // 返回 (金额, 币种, 错误)
}
```

暴露管理端点：`GET /v1/admin/balances`
- 遍历所有 ProviderConfig
- 如果厂商 Provider 实现了 `BalanceChecker`，则调用并返回

---

## 5. 实施优先级

| 优先级 | 任务 |
|---|---|
| P0 | 扩展 `models.yaml` + Registry 读取 providers 区块 |
| P0 | `internal/provider/factory.go` |
| P1 | `openai.go` 适配 OpenAI SSE 格式 |
| P1 | `anthropic.go` 适配 Anthropic SSE 格式 |
| P2 | `GET /v1/admin/balances` 端点 |
| P2 | 配置热更新（文件监听，不重启生效） |
