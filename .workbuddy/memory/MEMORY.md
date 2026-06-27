# QuantFlow 项目长期记忆

## 项目定位
QuantFlow Terminal — 双模式量化金融终端（彭博式面板 + 可视化工作流编排）。Go 1.22+/Wails v3 + Vue 3/TS + Python 3.12 gRPC + SQLite WAL。目标市场：A股>港股>美股>加密。AGPL-3.0。

## 已知关键问题（评审 2026-06-24 发现）
- **P0 金融正确性**：T+1 多标的回测失效(engine_cn.go:121)、OMS FillOrder 裁剪不一致(oms.go:122-161)、CacheKey map 序列化非确定(cache.go:32)、涨跌停完全缺失
- **P1**：无 look-ahead/survivorship bias 防护；ML 无验证集划分(tree_engine.py:110)；横截面因子经标准 RPC 失效(factor/engine.py:44)；Calmar 量纲不匹配(risk.go:94)；前端 ref<Map> 响应性失效(data.ts:57, workflow.ts:27)；wailsjs 绑定未生成，IPC 全 any
- 元数据不一致：go.mod 写 go 1.26.4（不存在）、config.yaml 版本 0.0.1 与 README 2026.6.19 不符

## 前端 Wails v3 注意事项
- Wails v3 不再有 `window.go` 全局对象，调用 Go Service 方法通过 `@wailsio/runtime` 的 `Call.ByName('main.App.Method', ...args)`
- `src/lib/wails.ts` 的 `setupWailsBridge()` 创建 `window.go` Proxy shim 提供 v2 兼容 — 面板代码 `(window as any).go.main.App.XXX` 均透明翻译
- `@wailsio/runtime` 无 TypeScript 声明，`src/types/wails-runtime.d.ts` 手动维护
- **Go embed 缓存陷阱**：`main.go` 用 `//go:embed all:frontend/dist` 打包前端。Go build cache 在 vite 改了文件名 hash 后可能不失效（cache key 漏判），导致 `go build` 打入旧 dist。症状："重新打包后 UI 没变"。修复：`go build -a`（强制全量重编译）或 `go clean -cache`。Taskfile `darwin:build` 已加 `-a` flag

## 工程规范
- 强制 Spec-Before-Code（docs/specs/）→ Plan（docs/superpowers/plans/）→ Execute 流程
- 关键代码（金融正确性/并发/安全/市场规则）强制 docstring
- SQLite 是唯一数据库，禁止引入 PG/Redis

## akshare 数据源注意事项
- akshare `macro_china_*` 返回中文业务列名（每端点不同），无统一 `value` 字段；`python/src/data/fincept/macro_cn.py:MACRO_CN_FIELDS` 是 endpoint→(date_col,value_col,unit,name_cn,category,polarity) 映射，新增端点须实测列名后加入
- 排序方向不统一：gdp/cpi/ppi/pmi/money_supply 逆序（最新在前），gdp_yearly/non_man_pmi/shibor_all/lpr 正序；`get_normalized()` 用智能日期排序统一，调用方不感知方向
- akshare 部分 endpoint 的 tqdm 进度条污染 stdout 破坏 JSON；`macro_cn.py` 顶部已设 `TQDM_DISABLE=1`
- FAST/SLOW/UNRELIABLE 端点分类见 `macro_cn.py` 文件头注释；`get_summary()` 仅并发拉 `FAST_CORE_ENDPOINTS`(9个)，SLOW 端点(60-120s)走 `macro_cn_indicator` 按需拉取
- `build/python/src/` 是构建副本，源码在 `python/src/`；改完记得同步或重建

## 宏观面板架构（2026-06-27 合并后）
- 统一面板：`registry.ts` 的 `macro`(研究分析>宏观经济) → `GovDataPanel.vue`；旧的 `gov-data`(另类数据) 和 `MacroPanel.vue` 已删除，勿再引用
- 三源切换：FRED(美国 GetEconomicIndicators+商品CL/NG) / CN(akshare macro_cn_summary) / BIS(dataflows目录)
- CN 信号语义（标准金融逻辑，`macro_cn.py` polarity 字段）：positive(value↑→bullish)=GDP/PMI/M2/贸易；negative(value↑→bearish)=CPI/PPI/失业率/Shibor；inverse(value↓→bullish)=LPR降息
- `get_normalized()`/`get_summary()` 已在后端算好 direction/change/signal，前端直接展示；勿在前端重复算 signal
