# iwencai Adapter — NL 语义搜索研报

## Motivation

iwencai（爱问财）是 SkillHub 2.0 的 NL 语义搜索 API，**唯一支持跨主题研报检索的数据源**。例如 "人形机器人 行星滚柱丝杠 2026" 这种跨标的、跨行业的主题搜索，东财 reportapi（按个股查）做不到。

SKILL.md 已有完整 Python 参考代码，但 QuantFlow 尚未实现 Go 适配器。用户已提供测试用 API Key，需要补齐：

- Go HTTP 适配器（`/v1/comprehensive/search` + `/v1/query2data`）
- API Key 通过环境变量配置（用户自行申请）
- 集成到 Research 层，为 NLP sentiment 和研报搜索提供文本源

## Design

### API 端点

| 端点 | 方法 | 用途 |
|------|------|------|
| `/v1/comprehensive/search` | POST | 语义搜索研报/公告/新闻 |
| `/v1/query2data` | POST | NL 数据查询（结构化字段） |

### 鉴权

```
Authorization: Bearer <IWENCAI_API_KEY>
X-Claw-Call-Type: normal
X-Claw-Skill-Id: report-search
X-Claw-Skill-Version: 2.0.0
X-Claw-Plugin-Id: none
X-Claw-Plugin-Version: none
X-Claw-Trace-Id: <32-char hex>
```

### 配置

```bash
export IWENCAI_API_KEY="sk-proj-..."   # 用户自行申请
export IWENCAI_BASE_URL="https://openapi.iwencai.com"  # 可选，有默认值
```

遵循现有模式：PolygonAdapter 用 `POLYGON_API_KEY`，TuShareAdapter 用 `TUSHARE_TOKEN`。

### 数据流

```
用户输入 "人形机器人 行星滚柱丝杠"
    → SymbolSearch / ResearchPanel
    → IwencaiAdapter.Search("人形机器人 行星滚柱丝杠", channel="report", size=50)
    → POST /v1/comprehensive/search
    → 去重 (同uid保留最高score)
    → 返回 []IwencaiArticle
    → 前端展示研报列表
```

### 新文件

- `internal/market/adapters/iwencai.go` — 适配器 + 类型定义
- `internal/market/adapters/iwencai_test.go` — 单元测试（需要 key）/ 离线测试

### 修改文件

- `app.go` — 创建 IwencaiAdapter，注入到 ResearchService
- `internal/research/analyst_estimates_service.go` — 可选：增加 `SearchReports(topic)` 方法
- `CHANGELOG.md`

## Acceptance Criteria

- [ ] `IwencaiAdapter` 实现 `market.Adapter` 接口（Name/IsAvailable/OHLCV/Quote）
- [ ] `IwencaiAdapter.Search("人形机器人", "report", 10)` 返回研报列表
- [ ] `IwencaiAdapter.Query("贵州茅台 ROE")` 返回结构化数据行
- [ ] 无 API Key 时 `IsAvailable()` 返回 false，优雅降级
- [ ] 去重逻辑：同一 uid 只保留最高 score
- [ ] Go test 通过（有 key 时跑 live test，无 key 时 skip）
- [ ] API Key 从环境变量读取，不硬编码

## Risks / Trade-offs

- **API Key 必选** — 唯一的付费/鉴权数据源。用户需自行去 iwencai.com/skillhub 申请
- **限流未知** — 上游未公布 QPS 限制，初期保守设 500ms 间隔
- **X-Claw Header 可能变更** — SkillHub 2.0 是较新协议，如果鉴权失败需检查 header 格式
- **返回格式不稳定** — `extra` 字段可能是 string 或 object，需要灵活解析
