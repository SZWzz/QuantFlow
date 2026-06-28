# Dragon Tiger + Limit Up/Down Panels (A-Share Signal EcoSystem)

## Motivation

A 股特有的龙虎榜和涨跌停是短线交易员最核心的两个信号源。龙虎榜揭示主力资金动向（营业部、机构、北向资金买卖），涨跌停监控提供封板强度、开板回封等盘口信号。两者深度耦合：涨停股票常出现在龙虎榜，龙虎榜数据可解释涨跌停原因。

## Design

### Data Flow

```
Go Backend                              Frontend
──────────────────────────────────────────────────
GetDailyDragonTiger(date, minNetBuy) → DragonTigerPanel
  └─ DragonTigerRaw {                    ├─ Daily Board (日榜单)
       date, stocks[]                    ├─ Per-symbol Detail (个股历史)
       └─ stock: code, name,             └─ symbol click → context.setGroupSymbol
            close, change_pct,
            net_buy_top5,                GetDragonTiger(symbol, endDate, lookBack)
            net_sell_top5,                  → symbol detail history
            department_buy_top5
     }

GetAbnormalStocks(market) → LimitUpDownPanel (filtered)
  └─ AbnormalStock[]                    ├─ Limit-Up list (涨停)
       └─ change_pct >= 9.8%            ├─ Limit-Down list (跌停)
          → mark as LimitUp             ├─ Nearly-limit (≥7%)
          → show queue/封单 info         └─ symbol click → context.setGroupSymbol
```

### DragonTigerPanel — Layout

```
┌─────────────────────────────────────────────┐
│ [Date Picker] [minNetBuy filter]  [refresh] │
├─────────────────────────────────────────────┤
│ Tab: 日榜单 | 个股历史                        │
├─────────────────────────────────────────────┤
│ 日榜单 tab:                                   │
│ ┌────┬────┬──────┬──────┬──────┬──────┐    │
│ │代码 │名称 │收盘价│涨跌幅 │净买入 │上榜原因│    │
│ ├────┼────┼──────┼──────┼──────┼──────┤    │
│ │点击→│ 联动 │ 切换 symbol                    │
│ └────┴────┴──────┴──────┴──────┴──────┘    │
│                                             │
│ 展开行: 买入TOP5 卖出TOP5 营业部TOP5           │
│ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│ │机构专用 1e8│ │华泰总部 5e7│ │...       │     │
│ │沪股通 2e7│ │...       │ │          │     │
│ └──────────┘ └──────────┘ └──────────┘     │
└─────────────────────────────────────────────┘
```

### LimitUpDownPanel — Layout

```
┌─────────────────────────────────────────────┐
│ [SH│SZ] [涨停│跌停│全部] [自动(30s)] [⟳]    │
├─────────────────────────────────────────────┤
│ 涨停列表:                    跌停列表:         │
│ ┌────┬────┬───┬───┬───┐  ┌────┬────┬───┬─┐ │
│ │代码│名称│价 │涨跌│封单│  │代码│名称│价 │...│ │
│ │000001│平安│...│+10%│1.2e│  │...               │ │
│ │ 点击→联动                                    │ │
│ └────┴────┴───┴───┴───┘  └────┴────┴───┴─┘ │
│ ◆ 涨停家数: 32  ◆ 跌停家数: 5                  │
└─────────────────────────────────────────────┘
```

### 跨面板联动

| 联动 | 机制 |
|------|------|
| 龙虎榜点击股票 → 打开行情详情 | `ctx.setGroupSymbol(groupId, code)` |
| 龙虎榜点击股票 → 涨跌停面板切换 | 监听 ctx group symbol 变化 |
| 涨跌停点击 → 行情详情/K线 | `ctx.setGroupSymbol(groupId, code)` |
| FundFlowPanel(龙虎榜 tab) ↔ DragonTigerPanel | 数据共享同一 Go API |

### Files

| File | Change |
|------|--------|
| `frontend/src/terminal/panels/DragonTigerPanel.vue` | **新增** |
| `frontend/src/terminal/panels/LimitUpDownPanel.vue` | **新增** |
| `frontend/src/terminal/panels/registry.ts` | 注册 `dragon-tiger`, `limit-up-down` |
| `frontend/src/lib/i18n/zh.ts` | 新增 keys |
| `frontend/src/lib/i18n/en.ts` | 新增 keys |

No Go changes needed.

## Acceptance Criteria

- [ ] 龙虎榜日榜单按日期查询，展示上榜股票 + 买卖 TOP5
- [ ] 龙虎榜个股历史切换，展示该股历史上榜记录
- [ ] 龙虎榜行点击 → 联动 symbolContext
- [ ] 涨跌停从 `GetAbnormalStocks` 数据中过滤涨幅≥9.8% / ≤-9.8%
- [ ] 涨跌停显示涨停/跌停家数统计
- [ ] 涨跌停行点击 → 联动 symbolContext
- [ ] 两个面板均支持自动刷新（30s），交易时间自动
- [ ] 骨架屏 loading 态
- [ ] i18n zh/en

## Risks

- A 股涨跌停阈值不同（科创板±20%、北交所±30%），需市场感知
- 龙虎榜数据 T+1 披露（收盘后），盘中仅昨日数据
