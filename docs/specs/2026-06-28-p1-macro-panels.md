# Macro-Analysis Panels (Sector Rotation + Economic Calendar)

## Motivation

板块轮动（RRG 相对强度分析）是量化研究员的核心分析框架：识别当前领涨/领跌板块、判断轮动方向（如何从成长切换为价值）、对比不同市场的行业强度。经济日历提供全球宏观事件的时间表，两者结合可回答"经济数据发布后哪些板块可能受益"。

## Design

### Data Flow

```
Go Backend                                    Frontend
────────────────────────────────────────────────────────────
GetIndustryRanks(topN) → SectorRotationPanel
  └─ []IndustryRank {                       ├─ RRG Scatter Chart (ECharts)
       name, change_pct                     │   x=JdK Ratio (动量)
     }                                      │   y=Rs Ratio (相对强度)
                                            │   4 quadrants: 领涨/转弱/领跌/转强
GetMarketOverview(CN).sectors               │
  → sector data                             ├─ Sector Strength Table
                                            │   名称 | 涨幅 | 动量 | 强度 | 信号
GetEconomicIndicators() → EconomicCalendar   │
  └─ []EconomicIndicator {                  ├─ Calendar Timeline
       series_id, name,                     │   日期 | 事件 | 前值 | 预期 | 实际
       last_value,                          │
       previous_value,                      │   高亮: 实际 vs 预期差异
       unit,                                │
       frequency                            │
     }                                      │
```

### SectorRotationPanel — Layout

```
┌─────────────────────────────────────────────┐
│ [CN│HK│US] [5d│20d│60d]          [refresh] │
├─────────────────────────────────────────────┤
│ ┌─ RRG Chart (ECharts scatter) ─────────┐  │
│ │  Improving ← → Leading               │  │
│ │    ↑ bank*    ↑ tech*   ↑ auto*      │  │
│ │  Lagging ← → Weakening                │  │
│ │    ↓ realty   ↓ energy                │  │
│ └───────────────────────────────────────┘  │
│                                             │
│ ┌─ Sector Strength Table ───────────────┐   │
│ │ 板块     │涨幅    │RS-Ratio│RS-Momentum │  │
│ │ 半导体   │+3.2%  │105.2  │102.1 →     │  │
│ │ 银行    │+1.5%  │98.5   │95.3  ↓     │  │
│ │ 房地产  │-2.1%  │92.1   │88.7  ↓     │  │
│ └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

### EconomicCalendarPanel — Layout

```
┌─────────────────────────────────────────────┐
│ [Filter: CN│US│All] [date picker] [⟳]      │
├─────────────────────────────────────────────┤
│ ┌─ Timeline ───────────────────────────┐   │
│ │ 06/28 周五                             │   │
│ │  ○ 20:30 US 核心PCE物价指数 前2.8% 预2.8% │   │
│ │    → [实际 2.7% ✓ 低于预期]              │   │
│ │  ○ 22:00 US 密歇根消费者信心 前69.1 预68.5│   │
│ │                                         │   │
│ │ 06/29 周六 (no events)                  │   │
│ │                                         │   │
│ │ 07/01 周一                              │   │
│ │  ○ 09:45 CN 财新制造业PMI 前51.7         │   │
│ └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

### 跨面板联动

| 联动 | 机制 |
|------|------|
| RRG 板块点击 → HeatmapPanel 高亮 | symbolContext（传入板块代码）|
| 经济日历数据 → 市场概况 | 用户手动切换（知识决策，非自动）|
| RRG 板块轮动信号 | 可输出到 NotifyPanel 预警 |

### Files

| File | Change |
|------|--------|
| `frontend/src/terminal/panels/SectorRotationPanel.vue` | **新增** |
| `frontend/src/terminal/panels/EconomicCalendarPanel.vue` | **新增** |
| `frontend/src/terminal/panels/registry.ts` | 注册 `sector-rotation`, `economic-calendar` |
| `frontend/src/lib/i18n/zh.ts` | 新增 keys |
| `frontend/src/lib/i18n/en.ts` | 新增 keys |

No Go backend changes (reuses existing `GetIndustryRanks`, `GetMarketOverview`, `GetEconomicIndicators`).

**Note on EconomicCalendar:** `GetEconomicIndicators()` 返回指标列表（FRED style），不是事件日历格式。前端需要将指标转换为"事件"展示格式。日历的"发布日期"信息使用 `frequency` + `last_value` 时间戳推导。原始数据中缺少的前值/预期/实际三级结构，使用 `previous_value` 作为前值、`last_value` 作为实际值。

## Acceptance Criteria

- [ ] RRG 散点图显示板块在 4 象限中的位置
- [ ] RRG 支持市场切换（CN/HK/US）和时间窗口（5/20/60 日）
- [ ] RRG 板块点击 → 联动 sector 上下文
- [ ] 板块强度表格含 RS-Ratio + RS-Momentum 列
- [ ] 经济日历按日期分组展示全球宏观事件
- [ ] 日历支持 CN/US/All 过滤器
- [ ] 日历显示事件的前值/预期/实际（有则显示）
- [ ] 骨架屏 loading
- [ ] i18n zh/en

## Risks

- RRG 需要最少 5 个数据点（日频），新上线板块无法展示
- 经济日历数据非原生事件日历，展示效果不如 Bloomberg 的 ECON 面板。未来可考虑接入专用事件日历 API。
- `GetMarketOverview().sectors` 可能只有当日数据，无历史序列。RRG 的 JdK 指数需要前 N 日 sector 数据。
