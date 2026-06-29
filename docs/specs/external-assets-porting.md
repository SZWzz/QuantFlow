# 外部项目可移植资产 — 最终 Spec

> **版本**: 0.3.0 | **日期**: 2026-06-27
> **调研对象**: 某终端项目 / 某 A 股项目 / 某 TDX 项目
> **方法**: 对比 QuantFlow 现有 40+ Go 适配器 + Python 数据层，识别真正缺失的资产

---

## 一、QuantFlow 现状核实（避免重复移植）

### Go 适配器层 — 已完整覆盖的数据源

| 类别 | 已有适配器 | 状态 |
|------|-----------|:--:|
| **A 股行情** | tencent, eastmoney, akshare, mootdx, baidu, sina, tushare, mac | ✅ 8 源回退链 |
| **北向资金** | `ths_northbound.go` | ✅ 同花顺 hsgtApi |
| **资金流** | `eastmoney_fundflow.go`, `eastmoney_capital.go` | ✅ |
| **板块/概念** | `eastmoney_concept.go` | ✅ |
| **iWenCai 选股** | `iwencai.go` | ✅ 已实现 |
| **热门/情绪** | `ths_hot.go`, `eastmoney_signals.go` | ✅ |
| **分析师一致预期** | `ths_consensus.go` | ✅ |
| **财报** | `sina_financials.go` | ✅ 新浪 |
| **巨潮公告** | `cninfo.go` | ✅ |
| **新闻** | `eastmoney_news.go`, `eastmoney_global_news.go`, `gdelt.go` | ✅ |
| **国会交易** | `congress_trades.go` | ✅ |
| **预测市场** | `polymarket.go` | ✅ |
| **卫星** | `satellite.go` | ✅ |
| **政府数据** | `govdata.go` | ✅ FRED 指标等 |
| **加密现货** | `binance.go`, `coingecko.go`, `okx.go`, `gateio.go` | ✅ 4 所 |
| **加密合约** | `binance_futures.go` | ✅ 刚新增 |
| **美股/全球** | `yahoo.go`, `finnhub.go`, `polygon.go` | ✅ 3 源 |

### Python 数据层 — 薄弱

| 服务 | 状态 |
|------|:--:|
| mootdx OHLCV/quote/minute | ✅ fetcher.py |
| **A 股财务三表详情** | ❌ 无 |
| **A 股基金 ETF 数据** | ❌ 无 |
| **A 股债券/可转债** | ❌ 无 |
| **A 股期货/期权** | ❌ 无 |
| **A 股龙虎榜/两融详情** | ❌ 无 |
| **宏观经济数据** | ❌ 无（govdata.go 仅基础） |
| **美股 SEC 深度** | ❌ 无 |
| **CCXT 跨所统一** | ❌ 仅 Binance 单所 |

---

## 二、确定缺失且有外部资产的领域

综合对比后，QuantFlow 真正缺失 + 外部可提供的资产如下：

### 类别 A：A 股深度金融数据（某终端项目 AKShare 脚本）

所有 32 个 某终端项目 AKShare 脚本都是**独立 Python 文件**，
无宿主耦合，pip install akshare 即可运行，输出 JSON/DataFrame。

| # | 脚本 | 行数 | 移植方式 | 成本 | 优先级 |
|---|------|:--:|---------|:--:|:--:|
| 1 | `akshare_stocks_financial.py` | 388 | 直接搬运 → `python/src/data/fincept/financials.py` | 1h | **P0** |
| 2 | `akshare_stocks_funds.py` | 448 | 直接搬运 → `python/src/data/fincept/fundflow.py` | 1h | **P0** |
| 3 | `akshare_stocks_board.py` | 393 | 合并到 `eastmoney_concept.go` 的 Python 增强 | — | 已有 Go 实现 |
| 4 | `akshare_stocks_margin.py` | 413 | 直接搬运 → `python/src/data/fincept/margin.py` | 1h | **P0** |
| 5 | `akshare_funds_expanded.py` | 594 | 直接搬运 → `python/src/data/fincept/funds.py` | 1.5h | P1 |
| 6 | `akshare_bonds.py` | 332 | 直接搬运 → `python/src/data/fincept/bonds.py` | 1h | P1 |
| 7 | `akshare_derivatives.py` | 392 | 直接搬运 → `python/src/data/fincept/derivatives.py` | 1h | P1 |
| 8 | `akshare_futures.py` | 464 | 合并到 derivatives.py | — | P1 |
| 9 | `akshare_economics_china.py` | 517 | 直接搬运 → `python/src/data/fincept/macro_cn.py` | 1h | P1 |
| 10 | `akshare_index.py` | 598 | 直接搬运 → `python/src/data/fincept/index.py` | 1h | P2 |

### 类别 B：全球经济数据（某终端项目 宏观经济脚本）

| # | 脚本 | 行数 | 移植方式 | 成本 | 优先级 |
|---|------|:--:|---------|:--:|:--:|
| 11 | `bis_data.py` (BIS) | 1445 | 直接搬运 → `python/src/data/fincept/macro_bis.py` | 1h | P1 |
| 12 | `wto_data.py` (WTO) | 889 | 直接搬运 → `python/src/data/fincept/macro_wto.py` | 1h | P1 |
| 13 | `eia_data.py` (能源) | 512 | 直接搬运 → `python/src/data/fincept/macro_eia.py` | 0.5h | P2 |
| 14 | `open_meteo_data.py` (天气) | 238 | 直接搬运 → `python/src/data/fincept/macro_climate.py` | 0.5h | P2 |

### 类别 C：美股 SEC 深度（某终端项目 EDGAR 脚本）

| # | 脚本 | 行数 | 移植方式 | 成本 | 优先级 |
|---|------|:--:|---------|:--:|:--:|
| 15 | `edgar/financials.py` | 251 | 直接搬运 → `python/src/data/fincept/sec_financials.py` | 1h | P1 |
| 16 | `edgar/forms_13f.py` | 268 | 直接搬运 → `python/src/data/fincept/sec_13f.py` | 0.5h | P1 |
| 17 | `edgar/forms_insider.py` | 234 | 直接搬运 → `python/src/data/fincept/sec_insider.py` | 0.5h | P1 |

### 类别 D：CCXT 加密统一层（某终端项目 exchange/ 脚本）

| # | 脚本 | 移植方式 | 成本 | 优先级 |
|---|------|---------|:--:|:--:|
| 18 | `exchange_client.py` + `fetch_*.py` (8 个) | 合并 → `python/src/data/ccxt_client.py` | 2h | **P0** |

### 类别 E：A 股特色分析（某 A 股项目 独有）

| # | 资产 | 移植方式 | 成本 | 优先级 |
|---|------|---------|:--:|:--:|
| 19 | **ST/*ST 风险预测** | 规则引擎 → `python/src/research/st_risk.py` | 4h | **P0** |
| 20 | **Alpha Zoo GTJA191** | 因子文件 → `python/src/factor/zoo/gtja191/` | 4h | **P1** |

### 类别 F：高级分析引擎（某终端项目 Analytics）

| # | 资产 | 移植方式 | 成本 | 优先级 |
|---|------|---------|:--:|:--:|
| 21 | **期权定价** (options.py) | 直接搬运 → `python/src/analytics/options.py` | 1h | **P0** |
| 22 | **组合优化** (portfolio_optimization.py) | 直接搬运 → `python/src/analytics/portfolio_opt.py` | 2h | P1 |
| 23 | **量化工具集** (quant_modules_3042.py) | 直接搬运 → `python/src/analytics/quant_tools.py` | 3h | P2 |

### 类别 G：TDX 依赖替换（某 TDX 项目）

| # | 资产 | 移植方式 | 成本 | 优先级 |
|---|------|---------|:--:|:--:|
| 24 | 某 TDX 项目 替换 mootdx | pip install 某 TDX 项目，修改 fetcher.py | 0.5h | P1 |

---

## 三、不推荐移植的资产（重复或低价值）

| 资产 | 来源 | 不移植原因 |
|------|------|-----------|
| iWenCai 适配器 | 某 A 股项目 | QuantFlow 已有 `iwencai.go` |
| 北向资金适配器 | 某 A 股项目 | QuantFlow 已有 `ths_northbound.go` |
| Alpha101/Qlib158 | 某 A 股项目 | 非 A 股专项因子，GTJA191 更合适 |
| DataHub 架构 | 某终端项目 | 概念复用，不涉及代码搬运 |
| C++ TradingTypes | 某终端项目 | 语言不匹配，作为设计参考 |
| Alpha Vantage 脚本 | 某终端项目 | QuantFlow 已有 yahoo/finnhub/polygon |
| yfinance_data.py | 某终端项目 | QuantFlow 已有 yahoo.go |
| open_meteo/卫星 | 某终端项目 | QuantFlow 已有 satellite.go |
| Databento | 某终端项目 | 付费 API，QuantFlow 定位免费工具 |
| MULTPL S&P 500 | 某终端项目 | 网页抓取不稳定，价值低 |
| agno_trading | 某终端项目 | 耦合过深，需大量适配 |

---

## 四、最终移植路线图

### Phase A — P0 立即移植（7 项，~10.5h）

```
python/src/data/fincept/          (新建目录)
├── financials.py    ← akshare_stocks_financial.py    (1h)   A股财务三表
├── fundflow.py      ← akshare_stocks_funds.py        (1h)   龙虎榜/两融/大宗
├── margin.py        ← akshare_stocks_margin.py       (1h)   两融详情
├── ccxt_client.py   ← exchange/ 8 脚本合并            (2h)   跨所加密统一层

python/src/research/
└── st_risk.py       ← 某 A 股项目 规则引擎           (4h)   ST退市预警

python/src/analytics/
└── options.py       ← 某终端项目 options.py      (1h)   BSM+二叉树+Greeks

frontend/src/terminal/panels/
└── 新增 FinancialsPanel / DerivativesPanel             (0.5h) 对应新数据的展示
```

### Phase B — P1 本轮移植（10 项，~13h）

```
python/src/data/fincept/
├── funds.py          ← akshare_funds_expanded.py       (1.5h) 基金ETF
├── bonds.py          ← akshare_bonds.py                (1h)   债券可转债
├── derivatives.py    ← akshare_derivatives+futures     (1h)   期货期权
├── macro_cn.py       ← akshare_economics_china.py      (1h)   宏观经济
├── macro_bis.py      ← bis_data.py                     (1h)   全球金融统计
├── macro_wto.py      ← wto_data.py                     (1h)   全球贸易数据
├── sec_financials.py ← edgar/financials.py             (1h)   SEC XBRL
├── sec_13f.py        ← edgar/forms_13f.py              (0.5h) 机构持仓
├── sec_insider.py    ← edgar/forms_insider.py          (0.5h) 内部人交易

python/src/factor/zoo/gtja191/
└── 191 个 A 股因子    ← 某 A 股项目 Alpha Zoo          (4h)   GTJA191

python/src/analytics/
└── portfolio_opt.py  ← 某终端项目 portfolio_opt    (2h)   组合优化

python/requirements.txt:
+ akshare, ccxt, edgartools 依赖                          (0.5h)
```

### Phase C — P2 后续纳入（4 项，~7.5h）

```
python/src/data/fincept/
├── index.py          ← akshare_index.py                (1h)   指数成分股
├── macro_eia.py      ← eia_data.py                     (0.5h) 能源数据
├── macro_climate.py  ← open_meteo_data.py              (0.5h) 气候数据

python/src/analytics/
└── quant_tools.py    ← quant_modules_3042.py            (3h)   量化工具集

python/fetcher.py:
+ 某 TDX 项目 替换 mootdx                                    (0.5h)

python/src/factor/zoo/
+ Alpha101 精选 50 个                                    (2h)
```

---

## 五、集成架构

```
QuantFlow 移植后架构：

Go Backend (internal/market/adapters/)  ← 不变，40+ 适配器保持
        │
        │ gRPC (已有 DataService)
        ▼
Python Sidecar (python/src/)
├── data/
│   ├── fetcher.py              ← 现有：mootdx wrapper
│   └── fincept/                 ← 新增：10+ 数据脚本
│       ├── financials.py       ← 某终端项目 搬运
│       ├── fundflow.py
│       ├── margin.py
│       ├── ccxt_client.py
│       ├── funds.py
│       ├── bonds.py
│       ├── derivatives.py
│       ├── macro_cn.py
│       ├── macro_bis.py
│       ├── macro_wto.py
│       ├── sec_financials.py
│       ├── sec_13f.py
│       └── sec_insider.py
├── factor/
│   └── zoo/
│       └── gtja191/             ← 新增：某 A 股项目 搬运
│           ├── factor_001.py
│           └── ... (191 个因子)
├── research/
│   └── st_risk.py               ← 新增：某 A 股项目 ST 预警
└── analytics/
    ├── options.py               ← 新增：某终端项目 期权
    ├── portfolio_opt.py         ← 新增：某终端项目 组合优化
    └── quant_tools.py           ← 新增：某终端项目 量化工具
```

---

## 六、移植可行性评估

| 维度 | 评估 |
|------|------|
| **许可证兼容** | 某终端项目 AGPL-3.0 ✅ / 某 A 股项目 MIT ✅ / 某 TDX 项目 MIT ✅ / QuantFlow AGPL-3.0 ✅ |
| **代码耦合度** | 所有标的脚本均为独立 Python 文件，输入参数 → 输出 JSON，无宿主框架耦合 |
| **依赖冲突** | akshare / edgartools / ccxt 均为 PyPI 标准包，无版本冲突风险 |
| **测试策略** | 每个搬运脚本附带 smoke test：`python -c "from fincept.financials import fetch; print(fetch('600519'))"` |
| **回退方案** | 所有新脚本放入独立 `fincept/` 子目录，Graft 而非 Modify，原模块零改动 |

---

*Spec 完*
