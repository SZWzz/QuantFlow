# 退市风险检测模块

## Motivation

现有 AuditPanel（财务审计面板）仅覆盖应收/商誉/现金流/负债率四项审计风险，缺少投资者最关心的**退市风险**判断。A 股 2024-2026 年退市制度改革后，财务类（营收+净利润组合、净资产为负）、交易类（面值<1元、市值<5亿）、规范类（资金占用、内控非标）和重大违法类四类退市标准日趋严格，2025 年已有 30 家公司退市，2026 年市值退市成为主流。港股和美股也有各自的退市规则体系。

用户需要在 AuditPanel 中一站式看到持仓标的的退市风险全景，而不必手动查阅交易所公告。

## Design

### Data Flow

```
AuditPanel.vue (existing)
  ├─ GetAuditFindings(symbol)     existing → Python analyzer
  ├─ GetFinancialAnalysis(symbol)  existing → Python analyzer
  └─ GetDelistingRisk(symbol)     NEW → Go pure computation
       │
       ├─ fetchFinancialJSON(symbol)   → 解析末期待摊: 营收/净利/扣非净利/净资产
       ├─ FetchStockInfo(symbol)       → 总市值/总股本/流通股本
       ├─ GetQuote(symbol)             → 最新价/成交量
       └─ boardDetection(symbol)       → 主板/创业板/科创板/北交所 (from symbol prefix)
```

### New/Modified Files

| File | Action | Description |
|------|--------|-------------|
| `app/internal/trading/delisting_risk.go` | **Create** | 规则引擎：数据提取 + 阈值判断 + 风险评级 |
| `app/app_research.go` | **Modify** | 新增 `GetDelistingRisk(symbol)` Wails 导出方法 |
| `frontend/src/terminal/panels/AuditPanel.vue` | **Modify** | 新增「退市风险」tab 标签页 |
| `frontend/src/terminal/panels/registry.ts` | **Modify** | 更新面板描述 |
| `docs/specs/2026-07-01-delisting-risk.md` | **Create** | 本 spec 文档 |

### Go Backend: `internal/trading/delisting_risk.go`

#### 符号前缀 → 板块映射

```go
func detectBoard(symbol string) string {
    switch {
    case strings.HasPrefix(symbol, "688"), strings.HasPrefix(symbol, "689"):
        return "科创板"
    case strings.HasPrefix(symbol, "300"), strings.HasPrefix(symbol, "301"):
        return "创业板"
    case strings.HasPrefix(symbol, "8"), strings.HasPrefix(symbol, "4"):
        return "北交所"
    case strings.HasPrefix(symbol, "60"), strings.HasPrefix(symbol, "00"):
        return "主板"
    default:
        return "未知"
    }
}
```

#### A 股规则映射

| 类别 | 指标 | 阈值 | 数据来源 |
|------|------|------|---------|
| 财务类 | 营收 < N 且扣非净利润为负 | 主板 3亿 / 双创 1亿 / 北交 1亿 | Sina 利润表末期 `营业总收入` + `净利润`/`扣非净利润` |
| 财务类 | 净资产为负 | < 0 | Sina 资产负债表末期 `归属于母公司所有者的权益合计` |
| 财务类 | 审计意见非标 | (数据暂缺) 标记"需人工核查" | — |
| 交易类 | 收盘价 < 1元 | 当前价预警（连续20日待历史K线） | GetQuote → `Last` |
| 交易类 | 总市值 < 阈值 | 主板 5亿 / 科创 3亿 / 北交 3亿 | FetchStockInfo → `MarketCap` |
| 交易类 | 成交量 < 阈值 | 主板120日<500万股（简化单日标记） | GetQuote → `Volume` |
| 规范类 | 资金占用 | (数据暂缺) 占位标记 | — |
| 重大违法 | 财务造假 | (数据暂缺) 占位标记 | — |
| 状态 | ST/\*ST | 股票名称含 "ST" 前缀 | 已有 `PriceLimitFor` 逻辑 |

#### 港股规则映射

| 指标 | 阈值 | 数据来源 |
|------|------|---------|
| 仙股化 | 收盘价 < 1 HKD | GetQuote → `Last` |
| 市值低迷 | 总市值 < 5亿 HKD | FetchStockInfo → `MarketCap` |
| 流动性枯竭 | 日换手率 < 0.02% | Volume / TotalShares |

#### 美股规则映射

| 指标 | 阈值 | 数据来源 |
|------|------|---------|
| NYSE 面值 | 连续30日收盘价 < 1 USD（当前价预警） | GetQuote → `Last` |
| NASDAQ 面值 | 同上 | GetQuote → `Last` |
| 市值 | < 5000万美元 | FetchStockInfo → `MarketCap` |
| 股东权益 | < 250万美元 | (数据暂缺) 标记待补充 |

#### 风险评分逻辑

每条指标返回三个状态之一，最终汇总 `overall_risk`：

| 状态 | 颜色 | 含义 | 条件 |
|------|------|------|------|
| `danger` | 红 | 已触及退市标准 | 净资产<0 或 市值<阈值 或 面值<1元 |
| `warn` | 黄 | 接近阈值需关注 | 营收接近退市线 或 股价距1元<30% 或 市值距阈值<20% |
| `safe` | 绿 | 远离阈值 | 全部远离 |

汇总规则：存在任一 `danger` → `high`；只有 `warn` 或更低 → `medium`；全 `safe` → `low`。

#### Go 函数签名

```go
type DelistingItem struct {
    Indicator string `json:"indicator"` // 指标名 e.g. "营收+净利润组合"
    Status    string `json:"status"`    // "safe" | "warn" | "danger"
    Current   string `json:"current"`   // 当前值 e.g. "营收12.5亿, 净利3.2亿"
    Threshold string `json:"threshold"` // 阈值 e.g. "营收<3亿且净利<0"
    Detail    string `json:"detail"`    // 说明 e.g. "远离退市线"
}

type DelistingCategory struct {
    Name  string          `json:"name"`  // e.g. "财务类退市"
    Level string          `json:"level"` // "green" | "yellow" | "red"
    Items []DelistingItem `json:"items"`
}

type DelistingRiskResult struct {
    Market      string             `json:"market"`       // "CN" | "HK" | "US"
    Board       string             `json:"board"`        // e.g. "主板"
    IsST        bool               `json:"is_st"`
    OverallRisk string             `json:"overall_risk"` // "low" | "medium" | "high"
    Categories  []DelistingCategory `json:"categories"`
    Summary     string             `json:"summary"`
}

func (a *App) GetDelistingRisk(symbol string) (*DelistingRiskResult, error)
```

#### Go 端实现要点

1. `GetDelistingRisk` 调用 `fetchFinancialJSON` 获取三表 JSON
2. 解析 JSON 提取末期的 `营业总收入`（或 `营业收入`）、`净利润`、`扣非净利润`（或 `归属于母公司所有者的净利润`）、`归属于母公司所有者权益合计`（或 `所有者权益合计`）
3. 调用 `a.sinaFinAdpt`（或 EastMoney）的 `FetchStockInfo` 获取总市值/总股本
4. 调用 `GetQuote` 获取最新价
5. 按符号前缀判定板块，计算对应阈值
6. 组装 `DelistingRiskResult` 返回

### Frontend: AuditPanel.vue

在现有面板新增一个 tab 标签页 "退市风险"：

```
┌─────────────────────────────────────────────┐
│ 财务审计  [财务健康] [审计发现] [退市风险]      │ ← 新增 tab
├─────────────────────────────────────────────┤
│                                             │
│  ┌───── Overall Risk ──────────────────┐    │
│  │            🟢 低风险                  │    │
│  └──────────────────────────────────────┘    │
│                                             │
│  ┌─ 财务类退市 ─────────────────────────┐    │
│  │ 🟢 营收+净利润组合  营收12.5亿 > 3亿  │    │
│  │ 🟢 净资产          净资产25.6亿 > 0   │    │
│  │ ⚪ 审计意见类型     (数据源待补充)     │    │
│  └──────────────────────────────────────┘    │
│                                             │
│  ┌─ 交易类退市 ─────────────────────────┐    │
│  │ 🟡 面值(收盘价)   1.32元  距1元仅24%  │    │
│  │ 🟢 总市值          68.5亿 >> 5亿      │    │
│  │ 🟢 成交量           正常              │    │
│  └──────────────────────────────────────┘    │
│                                             │
│  📋 综合: 财务指标正常，股价接近面值退市线   │
│                                            │
└─────────────────────────────────────────────┘
```

**实现要点：**
- 在 AuditPanel 的 tab 数组新增 `delisting` tab
- 并行调用 `GetDelistingRisk`（与现有的 `GetAuditFindings`/`GetFinancialAnalysis` 并行）
- 渲染分类卡片列表，状态灯用彩色圆点（`SignalBadge` 组件可复用）
- 空状态/错误状态与现有面板风格一致

### Error Handling

- 财务数据拉取失败：`Categories` 返回空数组，`Summary` 显示"财务数据暂不可用"
- 行情数据拉取失败：交易类条目标记 "数据暂缺"，不影响财务类判断
- Python sidecar 不可用：不影响退市风险检测（纯 Go 计算）

### Testing

- Go table-driven test for `detectBoard`: `"600519" → "主板"`, `"300750" → "创业板"`, `"00700" → "未知"`
- Go table-driven test for delisting risk: provide mock financial JSON + mock market data, verify risk flags
- Go test for market routing: `"600519" → CN rules`, `"00700" → HK rules`, `"AAPL" → US rules`

## Acceptance Criteria

- [ ] `GetDelistingRisk` 返回正确的退市风险分类（A/港股/美股）
- [ ] A 股营收+净利润组合阈值按板块区分（主板 3亿 / 双创 1亿 / 北交 1亿）
- [ ] A 股市值退市阈值按板块区分（主板 5亿 / 科创+北交 3亿）
- [ ] ST/\*ST 状态从股票名称正确推断
- [ ] AuditPanel 新增「退市风险」tab，渲染分类卡片 + 风险状态灯
- [ ] 财务数据拉取失败时不阻塞行情类判断（graceful degradation）
- [ ] 港股规则正确应用（仙股/市值/换手率）
- [ ] 美股规则正确应用（面值/市值）
- [ ] 无数据源的指标标记"待补充"而非错误崩溃
- [ ] Go 端全部规则通过 table-driven tests

## Risks / Trade-offs

- **审计意见类型无法获取**：Sina 财务数据不包含审计意见信息。目前标记为"数据源待补充"，未来可通过 EastMoney 公告接口或 Python 爬取年报PDF 提取
- **"连续20日"检查不完整**：当前仅检查最新价，无法回查 20 个交易日。可后续通过 K-line 数据补全（已有 OHLCV 数据），本阶段在详情中注明"当前价预警"
- **规范类/重大违法类数据缺失**：资金占用和财务造假认定依赖交易所公告，无法纯数据驱动。本阶段留空占位，后续可接入 Python 公告爬虫
- **市值退市标准差异**：港股以港元计价，美股以美元计价，汇率波动不影响阈值比较（各自货币单位独立判断）
