# [待开发/部分完成] AI/MCP/Skills 增强

> **Status**: PENDING — 后续开发
> **Proposal ref**: NEW_PROJECT_PROPOSAL.md §6 (AI 智能体系统)
> **Priority**: 🟡 中

## Motivation

FinceptTerminal 的 MCP (81 工具) + AstockPursue 的 89 Skills 尚未集成。当前 Agent 框架仅能调用少数内置 capabilities，无法充分发挥 LLM Agent 的价值。

## 缺失组件清单

### MCP Provider (`internal/ai/mcp.go`)

| 组件 | 状态 | 说明 |
|------|------|------|
| MCP 协议实现 | 📋 | Model Context Protocol 服务端 |
| ToolRegistry (81+) | 📋 | 注册所有 MCP 工具 |
| AuthManager | 📋 | MCP 认证级别 (none/auth/verified/subscribed) |
| MCPProviderNode | 📋 | 工作流节点封装 |

### Skills 知识库 (`resources/skills/`)

| 组件 | 状态 | 说明 |
|------|------|------|
| 89 个 Skill Markdown 文件 | 📋 | 量化分析/交易/研究等技能知识 |
| Skill Loader | 📋 | 加载+注入 LLM prompt |
| Skill 搜索/筛选 | 📋 | 前端 Skill 浏览器 |

### Agent Node 增强

| 增强 | 状态 | 说明 |
|------|------|------|
| 多 Agent 协作 | 📋 | Bull vs Bear 辩论模式 |
| 工作流节点→LLM tool | 📋 | 将工作流节点注册为 LLM 可调用 function |
| Agent → 工作流结果 | 📋 | LLM 输出结构化后传入下游节点 |
| Agent 执行可视化 | 📋 | 前端展示 Agent 思考步骤 |

### Agent Profiles 补齐 (37+)

| Profile | 优先级 | 说明 |
|---------|--------|------|
| warren_buffett (价值投资) | 🟡 | |
| ray_dalio (风险平价) | 🟡 | |
| jim_simons (量化) | 🟢 | |
| peter_lynch (成长投资) | 🟢 | |
| george_soros (宏观) | 🟢 | |
| ... 31 more | 🟢 | |

### Capability 补齐

| Capability | 优先级 | 说明 |
|------------|--------|------|
| `search_symbol` | 🔴 | 股票搜索 |
| `list_factors` | 🔴 | 因子查询 |
| `compute_factor` | 🔴 | 因子计算 |
| `run_backtest` | 🟡 | 回测执行 |
| `get_portfolio` | 🟡 | 组合查询 |
| `place_order` | 🟡 | 下单 |
| `get_news` | 🟡 | 新闻查询 |
| `sentiment_analysis` | 🟡 | 情绪分析 |
| `financials` | 🟡 | 财务数据 |
| `technical_indicators` | 🟢 | 技术指标 |

## 实现模式

```go
// MCP Provider
type MCPProvider struct {
    tools     map[string]MCPTool
    auth      *AuthManager
}

func (p *MCPProvider) HandleToolCall(ctx context.Context, name string, args json.RawMessage) (any, error) {
    // 路由到对应的 capability
    return p.capRegistry.Execute(ctx, name, args)
}

// Skills registry
type SkillRegistry struct {
    skills map[string]*Skill // YAML + Markdown
}

func (r *SkillRegistry) Load(path string) error { ... }
func (r *SkillRegistry) InjectIntoPrompt(context string, skills []string) string { ... }
```

## Acceptance Criteria

- [ ] MCP provider 可实现工具调用
- [ ] 89 Skills 文件可用（初期可 20 个 MVP）
- [ ] AgentNode 可调用工作流节点作为 tool
- [ ] AIChatPanel 显示 Agent 思考+工具调用步骤
- [ ] 37+ Agent profiles 可用（YAML）
- [ ] 现有测试通过

## 工作量估算

- MCP Provider: ~4 天
- Skills 知识库 (20 MVP): ~3 天
- Capability 补齐 (8): ~3 天
- Agent Node 增强: ~2 天
- Profiles 补齐: ~1 天
- 前端 Agent 可视化: ~2 天
- 测试: ~2 天
- **合计: ~17 天**

## Risks / Trade-offs

- MCP 协议仍在演进，接口可能变化
- Skills 内容质量依赖人工编写，初期可 LLM 生成
- Agent 调用工作流节点需考虑并发安全和递归风险
