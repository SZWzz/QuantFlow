# Geopolitics 地缘政治风险模块 — 设计文档

> **Status**: Design — 等待实施计划
> **Part of**: 另类数据模块，子项目 2/4
> **Priority**: 🔴 高

## Motivation

GDELT Project 是全球最大的开放新闻事件数据库，实时监控全球 100+ 语言、数万新闻源的报道。通过 GDELT DOC 2.0 API，可免费获取：

1. **话题覆盖量趋势** — 某个地缘话题被报道的频次变化（TimelineVol）
2. **情绪走势** — 话题报道的平均语气得分，-10（极度负面）到 +10（极度正面）（TimelineTone）
3. **风险信号提取** — 当某个话题同时出现"覆盖量激增 + 语气转负"时，生成风险信号

## Design

### Architecture

```
GDELT DOC 2.0 API (api.gdeltproject.org/api/v2/doc/doc)
    │  免费、无 API Key、返回 JSON
    │  mode=TimelineVol → 话题覆盖量时间序列
    │  mode=TimelineTone → 平均情绪得分时间序列
    │  format=json, timespan=7d/30d
    │
    ▼
GeopoliticsAdapter (internal/market/adapters/gdelt.go)
    │  接口: GeopoliticsAdapter interface
    │  FetchTopicVolume(topic, timespan) → []VolumePoint
    │  FetchTopicTone(topic, timespan) → []TonePoint
    │
    ▼
GeopoliticsService (internal/research/geopolitics_service.go)
    │  10 个预定义地缘话题 + TTL 缓存 (5min) + 风险评分
    │  GetTopicRisks() → []TopicRisk (所有话题的风险概况)
    │  GetTopicDetail(topicID) → 带历史的单个话题详情
    │  ExtractRiskSignals() → 检测覆盖量+情绪双重异常
    │
    ├──► GeopoliticsPanel (Vue 前端)
    │     ├── 话题卡片网格：10 个话题卡片，每卡显示 risk_level + 情绪分数
    │     ├── 话题详情页：情绪走势 ECharts + 近期文章列表
    │     └── 风险排序：高/中/低风险过滤
    │
    └──► geopolitics 工作流节点
          输入: topic (string) / region (string)
          输出: risk_signal (Signal) / risk_score (number) / tone (number)
```

### 10 个预定义地缘话题

| ID | 话题 | GDELT Query | 关联资产 |
|----|------|------------|---------|
| middle-east | 中东局势 | `"middle east" OR israel OR gaza OR iran OR "saudi arabia"` | 原油 |
| taiwan-strait | 台海紧张 | `"taiwan strait" OR "south china sea" OR taiwan OR "one china"` | A股/港股 |
| ukraine-war | 俄乌战争 | `"ukraine war" OR russia ukraine OR zelensky OR putin` | 能源/粮食 |
| trade-tariffs | 贸易关税 | `tariffs OR "trade war" OR "supply chain disruption" OR sanctions` | 全球 |
| north-korea | 朝鲜半岛 | `"north korea" OR kim OR pyongyang OR "missile launch"` | 韩国/日元 |
| fed-policy | 美联储政策 | `"federal reserve" OR fomc OR "rate hike" OR "rate cut" OR powell` | 美股/美元 |
| europe-energy | 欧洲能源 | `"europe energy" OR "natural gas" OR "energy crisis"` | 天然气 |
| terrorism | 恐怖主义 | `terrorism OR "terrorist attack" OR "security threat" OR extremism` | 全球 |
| china-economy | 中国经济 | `"china economy" OR "china gdp" OR "china property" OR "evergrande"` | A股/港股 |
| semiconductors | 半导体 | `semiconductor OR chips OR "chip ban" OR tsmc OR "export control"` | 科技股 |

### Data Model

```go
// GeopoliticsAdapter — 地缘政治数据专用接口
type GeopoliticsAdapter interface {
    Name() string
    IsAvailable(ctx context.Context) bool

    // FetchTopicVolume returns coverage volume time series for a topic.
    FetchTopicVolume(ctx context.Context, topicID string, timespan string) ([]VolumePoint, error)

    // FetchTopicTone returns average tone time series for a topic.
    FetchTopicTone(ctx context.Context, topicID string, timespan string) ([]TonePoint, error)
}

type VolumePoint struct {
    Date   string  `json:"date"`
    Value  float64 `json:"value"`  // article count or % of global coverage
    Query  string  `json:"query"`
}

type TonePoint struct {
    Date  string  `json:"date"`
    Tone  float64 `json:"tone"`  // -10 (extreme neg) to +10 (extreme pos)
    Query string  `json:"query"`
}

// TopicRisk represents the risk assessment for a single geopolitical topic.
type TopicRisk struct {
    ID          string  `json:"id"`
    Title       string  `json:"title"`
    TitleCN     string  `json:"title_cn"`
    RiskLevel   string  `json:"risk_level"`   // high / medium / low
    Tone        float64 `json:"tone"`          // current average tone
    ToneChange  float64 `json:"tone_change"`   // 7-day tone change
    VolChange   float64 `json:"vol_change"`    // 7-day volume change (%)
    Associated  string  `json:"associated"`     // assets affected
    UpdatedAt   int64   `json:"updated_at"`
}
```

### Frontend Panel

**GeopoliticsPanel** (`geopolitics`):

- **卡片网格**：10 个话题卡片，2 列 × 5 行布局
- **每张卡片**：中文标题 + 风险徽标（高🔴/中🟡/低🟢）+ 情绪分数 + 关联资产
- **点击展开**：情绪走势 ECharts（7 天）+ 覆盖量走势 + 文章预览
- **风险过滤**：全部 / 高风险 / 中风险 / 低风险
- **Mock 数据**：10 个话题配有模拟情绪数据

### Graceful Degradation

所有方法在 adapter nil 或 API 不可用时回退到 mock 数据。Mock 数据包含 10 个预定义话题的合理风险评分。

## Acceptance Criteria

- [ ] `GeopoliticsAdapter.FetchTopicVolume("taiwan-strait", "7d")` 返回覆盖量时间序列
- [ ] `GeopoliticsAdapter.FetchTopicTone("taiwan-strait", "7d")` 返回情绪时间序列
- [ ] `GeopoliticsService.GetTopicRisks()` 返回 10 个话题的风险概况
- [ ] `GeopoliticsService.ExtractRiskSignals()` 检测覆盖量+情绪双重异常
- [ ] Panel 渲染 10 个话题卡片 + 风险排序过滤
- [ ] Panel 显示 ECharts 情绪走势图（点击卡片展开）
- [ ] Panel 降级到 mock 数据（API 不可用时）
- [ ] Workflow node `geopolitics` 注册并可执行
- [ ] Node 输出 `risk_signal` + `risk_score` + `tone`
- [ ] `app.go` 导出 `GetGeopoliticsRisks()` + `GetGeopoliticsDetail()`
- [ ] `go vet ./...` 通过，Go 测试通过，前端测试通过

## Risks / Trade-offs

- **GDELT 国内访问** — 可能需要代理；API 不可用时走 mock
- **查询语法复杂** — 使用 URL 编码的布尔查询，已在话题配置中硬编码
- **TimelineVol 返回百分比** — 非绝对篇数，仅在同一时间段内有可比性
- **免费 API 无 SLA** — 响应时间可能 >5 秒，adapter timeout 设 30s
