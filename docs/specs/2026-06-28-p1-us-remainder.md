# 美股补缺: 期权链 + Wash Sale + 机构交易(暗池代理)

## Motivation

美股是第三大市场，目前只有基础行情/做空数据/财报日历面板。缺少三个工具：

1. **期权链** — 标准期权链（行权价×到期日矩阵），Finnhub 免费 API 可用
2. **Wash Sale 规则** — IRS 第 1091 条洗售亏损识别，纯 Go 逻辑无需外部 API
3. **机构交易** — SEC 13F/4 文件中的大额交易（作为暗池的免费替代），Finnhub SEC filings 可用

## Design

### 数据流

```
期权链:
  Finnhub GET /stock/option-chain?symbol=AAPL
    → FinnhubAdapter.FetchOptionChain(symbol)
      → App.GetUSOptionChain(symbol)
        → USOptionsPanel.vue

Wash Sale:
  OMS.GetTrades() + position data
    → internal/trading/wash_sale.go.CheckWashSale(symbol)
      → App.CheckWashSale(symbol)
        → WashSalePanel.vue

机构交易:
  Finnhub GET /stock/filings?symbol=AAPL
    → FinnhubAdapter.FetchSECFilings(symbol)
      → App.GetSECFilings(symbol)
        → DarkPoolPanel.vue (命名"机构交易")
```

### 修改文件

**Go Backend:**
- `internal/market/adapters/finnhub.go` — +`FetchOptionChain` + `FetchSECFilings`
- `internal/trading/wash_sale.go` — 新建文件
- `app_market.go` — +`GetUSOptionChain` + `GetSECFilings`
- `app_trading.go` — +`CheckWashSale`

**Frontend:**
- `USOptionsPanel.vue` — 期权链
- `WashSalePanel.vue` — wash sale
- `DarkPoolPanel.vue` — 机构交易
- registry.ts + i18n + CHANGELOG

### 面板设计

**USOptionsPanel**: 标准期权链布局（行权价左中右, 看涨/看跌 OI/成交量/隐含波动率/希腊字母）
**WashSalePanel**: 输入代码→扫描交易→标记洗售事件→显示受影响亏损+成本基础调整
**DarkPoolPanel**: SEC 13F/4 文件列表（机构/内部人交易，当作暗池活动的代理指标）

## Risks
- Finnhub 期权链免费套餐有速率限制 (60 req/min)
- Wash Sale 结论仅供参考，不构成税务建议
- 机构交易≠真正的暗池数据
