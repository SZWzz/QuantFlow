# [待开发] 另类数据模块

> **Status**: PENDING — 后续开发
> **Proposal ref**: NEW_PROJECT_PROPOSAL.md §4.1.6 (另类数据模块)
> **Priority**: 🟡 中

## Motivation

某终端项目的另类数据（预测市场/海事/地缘政治/卫星/政府）是区别于传统金融终端的关键差异化功能。目前 QuantFlow 完全未实现。

## 缺失组件清单

### Backend: 新数据适配器 + Workflow 节点

| 组件 | 状态 | 说明 |
|------|------|------|
| PredictionMarketAdapter | 📋 | Polymarket/Kalshi API 适配器 |
| MaritimeAdapter | 📋 | 海事/船舶 AIS 数据 |
| GeopoliticsAdapter | 📋 | 地缘政治事件数据 |
| SatelliteAdapter | 📋 | NASA/Sentinel 卫星数据 |
| GovDataAdapter | 📋 | 政府数据 (SEC/EDGAR/央行) |

### Workflow Nodes

| 节点 | 说明 |
|------|------|
| `prediction_market` | 预测市场数据→概率信号 |
| `maritime_data` | 航运/大宗商品供需信号 |
| `geopolitics_data` | 地缘政治事件→风险信号 |
| `satellite_data` | 卫星指标 (NDVI/基建) |
| `gov_data` | 政府经济数据 |

### Frontend Panels

| 面板 | 说明 |
|------|------|
| PredictionMarketPanel | 预测市场赔率/概率仪表盘 |
| MaritimePanel | 航运跟踪/大宗商品面板 |
| GeopoliticsPanel | 地缘政治热力图 |
| SatellitePanel | 卫星数据分析面板 |
| GovDataPanel | 政府经济数据浏览 |

## Data Flow

```
[Market Adapter] → [MarketDataHub] → [Panel/Frontend]
                                    → [Workflow Node → Downstream]
```

所有另类数据适配器遵循 `internal/market/adapters/adapter.go` 的 Adapter 接口。

## Acceptance Criteria

- [ ] 5 个另类数据适配器实现 `DataAdapter` 接口
- [ ] 5 个工作流节点注册到 NodeRegistry
- [ ] 5 个前端面板可渲染（初期 mock 数据）
- [ ] 现有测试通过

## 工作量估算

- 适配器 (5个): ~5 天
- 节点 (5个): ~2 天
- 面板 (5个): ~5 天
- 测试: ~2 天
- **合计: ~14 天**

## Risks / Trade-offs

- Polymarket/Kalshi API 在部分国家受限
- 海事 AIS 数据通常需要商业订阅
- 卫星数据处理复杂（Python 端）
