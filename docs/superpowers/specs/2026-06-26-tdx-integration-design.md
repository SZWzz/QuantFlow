# 通达信能力集成 — 设计文档

> 基于某外部项目（MIT 协议）补齐 QuantFlow 在 A 股技术分析、数据深度、策略生态上的短板。

## 现状 vs 目标

| 领域 | 现状 | 目标 |
|------|------|------|
| 技术指标 | 15 个 workflow nodes | **34 个**（补齐 KDJ、DMI、ATR、WR、CCI、BIAS、OBV、MFI、SAR、VWAP、捉妖大师等 19 个） |
| 缠论分析 | 无 | **完整缠论管道**（分型→笔→中枢→线段→买卖点→背驰→多级别联立） |
| K 线复权 | 无 | **前复权/后复权/不复权** |
| MAC 协议 | 无 | **板块排名、资金流向、集合竞价、异动监控、分时多日** |
| 内置策略 | 0 个 | **16 个开箱即用策略模板** |
| 选股扫描 | 无 | **全市场 4800+ A 股并发扫描、按绩效排名** |
| 回测增强 | 基础 | **滑点模型、执行仿真、组合回测、批量对比** |
| 因子分析 | 仅计算 | **IC 分析、分层回测、衰减分析、预处理** |
| 离线数据 | 无 | **本地数据同步、通达信文件读写** |

---

## 分层设计

### Phase 1 — 技术指标补齐 + K 线复权（1-2 天）

**定位**：最小改动、最大效果。直接在 Python sidecar 中新增指标计算模块。

**核心交付**：
- Python 新增 `python/src/indicators/` 包，移植某外部项目的 34 个指标算法
- 新增 gRPC 服务 `ComputeIndicators`，接收 OHLCV DataFrame + 指标名列表，返回多指标结果
- Go 后端新增 19 个 workflow nodes（每个指标一个 node）
- mootdx adapter 新增复权参数 `fqfactor`（前复权/后复权/不复权）

**改动范围**：
- Python：`indicators/` 包（19 个指标文件） + proto 定义 + server 注册
- Go：19 个 node 文件（`internal/workflow/nodes/indicator_xxx.go`）
- Go：`internal/market/adapters/mootdx.go` 加复权

### Phase 2 — 缠论分析（1 天）

**定位**：最大差异化能力。Python sidecar 新增 `chanlun/` 模块。

**核心交付**：
- K 线包含处理 → 分型识别 → 笔 → 中枢 → 线段 → 买卖点 → 背驰 → 多级别联立
- Go 端新增 `ChanlunNode` workflow node
- 前端新增 `ChanlunPanel.vue` 展示缠论结果

### Phase 3 — 内置策略库 + 选股扫描（1-2 天）

**定位**：用户体验跃升。零基础上手。

**核心交付**：
- `strategies/` 目录下 16 个策略文件（Python）
- Go 端 `StrategyScanNode` — 全市场并发选股
- Go 端 `BatchBacktestNode` — 批量回测 + 排名
- 前端新增选股结果面板

### Phase 4 — 回测增强 + 因子分析（1 天）

**核心交付**：
- 滑点模型（根号冲击/成交量比例）
- TWAP/VWAP 执行仿真
- 组合回测（多标的资金池）
- 因子 IC 分析、分层回测、衰减分析、预处理

### Phase 5 — MAC 协议 + 离线数据（1-2 天）

**核心交付**：
- Go 端新增 `mac_adapter.go` 直接 TCP 连接 TDX MAC 协议
- 板块排名、资金流向、集合竞价、异动监控
- 离线数据同步：全市场日线下载 + 本地 `.day` 文件读写

---

## 验证标准

1. 34 个指标全部可通过 workflow node 调用
2. 缠论分析输出完整的分型/笔/中枢/买卖点
3. K 线复权后回测结果与不复权有显著差异
4. 选股扫描可在 60s 内完成 4800+ A 股
5. 16 个策略可直接 `python strategy_file.py --symbol 600519` 运行
6. 回测包含滑点和执行仿真
7. npm run build + go build 通过
