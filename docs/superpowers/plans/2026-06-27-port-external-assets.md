# 外部资产移植实施计划

> **For agentic workers:** All tasks are independent file-copy operations. Dispatch in parallel waves (P0 → P1 → P2).

**Goal:** 从 某终端项目 + 某 A 股项目 移植 19 项 Python 数据/分析资产到 QuantFlow

**Architecture:** 纯 Python 文件搬运 → `python/src/data/fincept/` + `python/src/analytics/` + `python/src/research/` + `python/src/factor/zoo/gtja191/`。每个脚本带回后加一条 smoke test。

**Spec:** docs/specs/external-assets-porting.md

**Tech Stack:** Python 3.12, akshare, ccxt, edgartools, numpy, pandas

## Global Constraints

- 源文件路径: 某终端项目 = `/Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/`，某 A 股项目 = `/Volumes/etx/coding/rebuild/某 A 股项目/`
- 目标目录: `/Volumes/etx/coding/rebuild/quantflow/python/src/`
- 不修改源文件的业务逻辑，只做 import 路径适配
- 每项搬运后运行 `python -c "from ... import ...; print('OK')"` 验证
- 所有依赖添加到 `python/requirements.txt`

---

## Phase A — P0 立即移植（6 项，~10h）

### Task 1: A 股财务三表数据

**Files:**
- Create: `python/src/data/fincept/__init__.py`
- Create: `python/src/data/fincept/financials.py`
- Source: `某终端项目/fincept-qt/scripts/akshare_stocks_financial.py`
- Copy: `某终端项目/fincept-qt/scripts/akshare_company_info.py` → `python/src/data/fincept/company_info.py`

**Step 1:** 创建 fincept 包目录
```bash
mkdir -p /Volumes/etx/coding/rebuild/quantflow/python/src/data/fincept
```

**Step 2:** 复制文件并适配 import
```bash
cp /Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/akshare_stocks_financial.py /Volumes/etx/coding/rebuild/quantflow/python/src/data/fincept/financials.py
cp /Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/akshare_company_info.py /Volumes/etx/coding/rebuild/quantflow/python/src/data/fincept/company_info.py
```

**Step 3:** Smoke test
```bash
cd /Volumes/etx/coding/rebuild/quantflow/python && python -c "from src.data.fincept.financials import *; print('financials OK')"
```

### Task 2: A 股资金流 + 两融数据

**Files:**
- Create: `python/src/data/fincept/fundflow.py`
- Create: `python/src/data/fincept/margin.py`
- Source: `某终端项目/fincept-qt/scripts/akshare_stocks_funds.py`, `akshare_stocks_margin.py`

**Step 1:** 复制
```bash
cp /Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/akshare_stocks_funds.py /Volumes/etx/coding/rebuild/quantflow/python/src/data/fincept/fundflow.py
cp /Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/akshare_stocks_margin.py /Volumes/etx/coding/rebuild/quantflow/python/src/data/fincept/margin.py
```

**Step 2:** Smoke test
```bash
cd /Volumes/etx/coding/rebuild/quantflow/python && python -c "from src.data.fincept.fundflow import *; print('fundflow OK')"
```

### Task 3: CCXT 跨所加密统一层

**Files:**
- Create: `python/src/data/fincept/ccxt_client.py`
- Source: `某终端项目/fincept-qt/scripts/exchange/` (合并核心 8 文件)

**Step 1:** 复制 + 合并
```bash
# 合并 exchange_client + fetch_ohlcv + fetch_ticker + fetch_orderbook
# + fetch_funding_rate + fetch_open_interest + fetch_markets + list_exchanges
# 为一个统一的 ccxt_client.py (~400 行)
```

**Step 2:** Smoke test
```bash
cd /Volumes/etx/coding/rebuild/quantflow/python && python -c "from src.data.fincept.ccxt_client import *; print('ccxt OK')"
```

**Step 3:** 添加依赖
```bash
echo "ccxt>=4.0" >> /Volumes/etx/coding/rebuild/quantflow/python/requirements.txt
```

### Task 4: ST/*ST 风险预测规则引擎

**Files:**
- Create: `python/src/research/__init__.py`
- Create: `python/src/research/st_risk.py`
- Source: `某 A 股项目/services/python/src/skills/ashare-pre-st-filter/SKILL.md` (规则) + 自行实现

**Step 1:** 基于 SKILL.md 中的 4R+3E 规则框架实现 `st_risk.py`（~300 行）

核心函数：
```python
def predict_st_risk(symbol: str) -> dict:
    """Return {'risk_level': 'high'|'medium'|'low', 'confidence': float, 'reasons': list}"""
    pass
```

**Step 2:** Smoke test
```bash
cd /Volumes/etx/coding/rebuild/quantflow/python && python -c "from src.research.st_risk import predict_st_risk; print('st_risk OK')"
```

### Task 5: 期权定价引擎

**Files:**
- Create: `python/src/analytics/__init__.py`
- Create: `python/src/analytics/options.py`
- Source: `某终端项目/fincept-qt/scripts/Analytics/derivatives/options.py` (268 行)

**Step 1:** 直接复制
```bash
cp /Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/Analytics/derivatives/options.py /Volumes/etx/coding/rebuild/quantflow/python/src/analytics/options.py
```

**Step 2:** Smoke test
```bash
cd /Volumes/etx/coding/rebuild/quantflow/python && python -c "from src.analytics.options import black_scholes_merton, delta, gamma, vega, implied_volatility; print('options OK')"
```

### Task 6: 前端面板 — FinancialsPanel + OptionsPanel

**Files:**
- Create: `frontend/src/terminal/panels/FinancialsPanel.vue`
- Create: `frontend/src/terminal/panels/OptionsPanel.vue`

**Step 1:** 创建基础的 FinancialsPanel.vue（展示财务数据表格）
**Step 2:** 创建基础的 OptionsPanel.vue（展示期权希腊值）
**Step 3:** 注册到面板系统
**Step 4:** `npx vite build` 验证

---

## Phase B — P1 本轮移植（10 项，~13h）

### Task 7-16: 数据脚本批量搬运

所有脚本从 某终端项目 直接复制，无需修改业务逻辑：

| Task | 目标文件 | 源文件 |
|------|---------|--------|
| 7 | `data/fincept/funds.py` | `akshare_funds_expanded.py` |
| 8 | `data/fincept/bonds.py` | `akshare_bonds.py` |
| 9 | `data/fincept/derivatives.py` | `akshare_derivatives.py` + `akshare_futures.py` 合并 |
| 10 | `data/fincept/macro_cn.py` | `akshare_economics_china.py` |
| 11 | `data/fincept/macro_bis.py` | `bis_data.py` (需适配 import: `aiohttp`→`requests`) |
| 12 | `data/fincept/macro_wto.py` | `wto_data.py` |
| 13 | `data/fincept/sec_financials.py` | `mcp/edgar/financials.py` |
| 14 | `data/fincept/sec_13f.py` | `mcp/edgar/forms_13f.py` |
| 15 | `data/fincept/sec_insider.py` | `mcp/edgar/forms_insider.py` |
| 16 | `factor/zoo/gtja191/` | `某 A 股项目/services/python/src/factors/zoo/gtja191/` |

每项的验证命令：
```bash
cd /Volumes/etx/coding/rebuild/quantflow/python
python -c "from src.data.fincept.funds import *; print('OK')"
# ... 依次验证每个模块
```

依赖添加：
```bash
echo "akshare>=1.14" >> requirements.txt
echo "edgartools>=0.5" >> requirements.txt
```

### Task 17: 组合优化引擎

**Files:**
- Create: `python/src/analytics/portfolio_opt.py`
- Source: `某终端项目/fincept-qt/scripts/Analytics/portfolioManagement/portfolio_optimization.py`

```bash
cp /Volumes/etx/coding/rebuild/某终端项目/fincept-qt/scripts/Analytics/portfolioManagement/portfolio_optimization.py /Volumes/etx/coding/rebuild/quantflow/python/src/analytics/portfolio_opt.py
```

---

## Phase C — P2 补充移植（3 项，~5h）

| Task | 目标文件 | 源文件 |
|------|---------|--------|
| 18 | `data/fincept/index.py` | `akshare_index.py` |
| 19 | `data/fincept/macro_eia.py` | `eia_data.py` |
| 20 | `analytics/quant_tools.py` | `Analytics/quant/quant_modules_3042.py` |

---

## 依赖汇总（一次性添加）

```txt
# python/requirements.txt 追加：
akshare>=1.14.0
ccxt>=4.4.0
edgartools>=0.5.0
```

---

## 全量验证

```bash
# Step 1: 所有模块导入
cd /Volumes/etx/coding/rebuild/quantflow/python
python -c "
from src.data.fincept.financials import *
from src.data.fincept.fundflow import *
from src.data.fincept.margin import *
from src.data.fincept.ccxt_client import *
from src.data.fincept.funds import *
from src.data.fincept.bonds import *
from src.data.fincept.derivatives import *
from src.data.fincept.macro_cn import *
from src.data.fincept.macro_bis import *
from src.data.fincept.macro_wto import *
from src.data.fincept.sec_financials import *
from src.data.fincept.sec_13f import *
from src.data.fincept.sec_insider import *
from src.research.st_risk import *
from src.analytics.options import *
from src.analytics.portfolio_opt import *
print('ALL 16 MODULES OK')
"

# Step 2: 前端构建
cd frontend && npx vite build 2>&1 | tail -3

# Step 3: Go 构建（确认无影响）
cd .. && go build -o /dev/null . 2>&1 | grep -v "ld: warning" | head -3
```

---

## 执行策略

**Wave 1 (P0)**: Dispatch 3 parallel agents:
- Agent A: Tasks 1+2 (financials + fundflow + margin) — 文件复制
- Agent B: Tasks 3+5 (ccxt_client + options) — 文件复制
- Agent C: Task 4 (st_risk) — 编码 + Task 6 (前端面板) — 编码

**Wave 2 (P1)**: Dispatch 2 parallel agents:
- Agent D: Tasks 7-15 (10 个数据脚本批量搬运)
- Agent E: Tasks 16-17 (GTJA191 因子 + portfolio_opt)

**Wave 3 (P2)**: 1 agent:
- Agent F: Tasks 18-20 (index + eia + quant_tools)

**Wave 4**: 全量验证

*Plan 完*
