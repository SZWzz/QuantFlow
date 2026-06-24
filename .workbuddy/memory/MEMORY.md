# QuantFlow 项目长期记忆

## 项目定位
QuantFlow Terminal — 双模式量化金融终端（彭博式面板 + 可视化工作流编排）。Go 1.22+/Wails v3 + Vue 3/TS + Python 3.12 gRPC + SQLite WAL。目标市场：A股>港股>美股>加密。AGPL-3.0。

## 已知关键问题（评审 2026-06-24 发现）
- **P0 金融正确性**：T+1 多标的回测失效(engine_cn.go:121)、OMS FillOrder 裁剪不一致(oms.go:122-161)、CacheKey map 序列化非确定(cache.go:32)、涨跌停完全缺失
- **P1**：无 look-ahead/survivorship bias 防护；ML 无验证集划分(tree_engine.py:110)；横截面因子经标准 RPC 失效(factor/engine.py:44)；Calmar 量纲不匹配(risk.go:94)；前端 ref<Map> 响应性失效(data.ts:57, workflow.ts:27)；wailsjs 绑定未生成，IPC 全 any
- 元数据不一致：go.mod 写 go 1.26.4（不存在）、config.yaml 版本 0.0.1 与 README 2026.6.19 不符

## 工程规范
- 强制 Spec-Before-Code（docs/specs/）→ Plan（docs/superpowers/plans/）→ Execute 流程
- 关键代码（金融正确性/并发/安全/市场规则）强制 docstring
- SQLite 是唯一数据库，禁止引入 PG/Redis
