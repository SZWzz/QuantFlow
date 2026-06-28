# HK Stock Connect Panel (港股通面板)

## Motivation

港股通是内地投资者参与香港市场的核心通道。北上资金（北向）的流入流出是 A 股的重要信号，南下资金（南向）的流向反映内地资金对港股的配置意愿。当前量化终端缺少一个专用的港股通仪表盘来展示：

1. 北上资金实时分时流向（沪股通+深股通）
2. 南下资金实时流向
3. 每日额度余额
4. 历史累计净买入趋势

## Design

### Data Flow

```
Go Backend                                  Frontend
────────────────────────────────────────────────────────
GetNorthboundFlow() → HKConnectPanel
  └─ {                                    ├─ Minute Flow Chart (分时)
       minute_flow: [{                     ├─ Daily History Table
         time, sh_net, sz_net, total       ├─ Stats Cards (今日/累计)
       }],
       history: [{
         date, sh_net, sz_net, total_net,
         sh_cum, sz_cum
       }]
     }
```

### Layout

```
┌─────────────────────────────────────────────┐
│ Tab: 北向资金 | 南向资金 | 额度概览  [⟳]      │
├─────────────────────────────────────────────┤
│ 北向资金 tab:                                 │
│ ┌─ Stats Cards ──────────────────────────┐ │
│ │ 今日沪股通: +45.2亿  │ 今日深股通: +28.7亿  │ │
│ │ 合计: +73.9亿        │ 累计: +18,234亿     │ │
│ └────────────────────────────────────────┘ │
│                                             │
│ ┌─ Minute Flow Chart (ECharts area) ────┐   │
│ │  [line chart: 沪股通/深股通/合计 分时]  │   │
│ │  (x=time, y=net flow in 亿)           │   │
│ └────────────────────────────────────────┘  │
│                                             │
│ ┌─ Daily History Table ─────────────────┐   │
│ │ 日期  │沪股通│深股通│合计│累计    │    │   │
│ │ 06-27 │+12  │+8   │+20 │+18234  │    │   │
│ │ 06-26 │+5   │-3   │+2  │+18214  │    │   │
│ └────────────────────────────────────────┘  │
├─────────────────────────────────────────────┤
│ 额度概览 tab:                                 │
│ 沪股通: 剩余 345亿/520亿 (66%)                │
│ 深股通: 剩余 412亿/520亿 (79%)                │
└─────────────────────────────────────────────┘
```

### 跨面板联动

| 联动 | 机制 |
|------|------|
| HKConnect 北上资金异动 → MarketOverview CN 行情 | 用户手动切换 market tab |
| HKConnect symbol 搜索（港股）→ 行情详情 | symbolContext |

### Files

| File | Change |
|------|--------|
| `frontend/src/terminal/panels/HKConnectPanel.vue` | **新增** |
| `frontend/src/terminal/panels/registry.ts` | 注册 `hk-connect` |
| `frontend/src/lib/i18n/zh.ts` | 新增 keys |
| `frontend/src/lib/i18n/en.ts` | 新增 keys |

No Go backend changes (uses existing `GetNorthboundFlow()`).

**Note:** `GetNorthboundFlow()` returns `map[string]interface{}` with `minute_flow` and `history`. 当前仅支持北向。南向数据待未来扩展 adapter。

## Acceptance Criteria

- [ ] 显示北向资金今日流入（沪/深/合计）+ 累计净买入
- [ ] 分时走势 ECharts 面积图
- [ ] 每日历史表格（最近 60 个交易日）
- [ ] 额度概览 tab（显示沪/深股通余额）
- [ ] 自动刷新（60s）
- [ ] 骨架屏 loading 态
- [ ] i18n zh/en

## Risks

- 南向数据当前无 adapter，额度数据也需从 GetNorthboundFlow 中推导
- 分时数据可能只有当日，历史表格需从 daily history 获取
