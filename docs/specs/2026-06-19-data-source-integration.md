# QuantFlow 全栈数据源整合 Spec

## Motivation

SKILL.md（`e:\coding\quantflow\SKILL.md`）来自 [a-stock-data](https://github.com/simonlin1212/a-stock-data) 项目，覆盖 A 股 **七层数据源、27 个验证端点**，全部免费（仅 iwencai 需 API Key），且包含完整的防封策略。当前 QuantFlow 只有行情层适配器（OHLCV/Quote），缺失 6/7 层数据源。最重要的是：**NLP 情绪分析管道已完整实现但无文本输入源**——所有 `GetSentiment()` 调用都传空字符串，始终返回 neutral mock 数据。

### 为什么要整合

1. **NLP 数据源缺失是阻塞性问题** — 情绪分析所有测试通过但功能为"空壳"
2. **SKILL.md 端点已经社区批量验证** — 2026-06 实测，含限流阈值和失效替换记录
3. **现有适配器模式已成熟** — 扩展新适配器只需实现接口方法 + 注册即可
4. **A 股数据源全部免费无 key** — 腾讯/通达信/东财/新浪/同花顺/巨潮，零 API 费用

## Design

### 七层数据源 → QuantFlow 适配器映射

#### Layer 1: 行情层（✅ 已实现无需新增）

mootdx K线/盘口 | 腾讯财经 PE/PB/市值 | 百度股市通 K线+MA | 东财 real-time | 新浪实时价

#### Layer 5: 新闻层（🔴 Phase 1 — NLP 文本来源）

| 数据源 | 协议 | 用途 |
|--------|------|------|
| 东财个股新闻 | HTTP JSONP (search-api-web) | 个股新闻标题+正文 → NLP sentiment |
| 东财全球资讯 (7×24) | HTTP JSON (np-weblist) | 全市场财经快讯 → 全局情绪 |

**数据流**:
```
GetSentiment(symbol)
    → 查缓存 (sentiment_cache)
    → NewsAdapter.FetchStockNews(symbol, limit=5)
    → 拼接文本: strings.Join(articles, "\n")
    → Python gRPC SentimentService(text_content=文本)
    → 写缓存 SaveSentiment(output)
    → 返回 SentimentOutput
```

**新接口**:
```go
// internal/market/adapters/news_adapter.go
type NewsArticle struct {
    Symbol  string
    Title   string
    Content string
    Time    string
    Source  string
}

type NewsAdapter interface {
    Name() string
    IsAvailable(ctx context.Context) bool
    FetchStockNews(ctx context.Context, symbol string, limit int) ([]NewsArticle, error)
}
```

#### Layer 3+4+6: 信号+资金面+基础数据层（📋 Phase 2）

同花顺热点/北向资金 | 东财概念板块/资金流/龙虎榜/解禁/行业排名 | 融资融券/大宗交易/股东户数/分红 | 新浪财报三表

#### Layer 2+7: 研报+公告层（📋 Phase 3）

东财研报 API | 同花顺一致预期 EPS | 巨潮公告全文 | 新浪财报三表

### 防封策略

所有东财域名请求统一走全局限流器（QPS ≤ 2，500ms 基础间隔 + 200ms 随机抖动），UA + Referer 完整头部，HTTP Keep-Alive 会话复用。

## Implementation Plan

### Phase 1: 新闻层 (Layer 5) — 🔴 最高优先，解锁 NLP

**新增文件 (7 个)**:
- `internal/market/adapters/news_adapter.go` — NewsArticle 类型 + NewsAdapter 接口
- `internal/market/adapters/eastmoney_news.go` — 东财个股新闻 (JSONP)
- `internal/market/adapters/eastmoney_news_test.go`
- `internal/market/adapters/eastmoney_global_news.go` — 东财全球资讯 (7×24)
- `internal/market/adapters/eastmoney_global_news_test.go`
- `internal/workflow/nodes/news_fetcher.go` — NewsFetcherNode
- `internal/workflow/nodes/news_fetcher_test.go`

**修改现有文件**:
- `app.go` — `GetSentiment()` 集成新闻获取；`GetStockResearch()` 同上
- `internal/research/sentiment_engine.go` — 当 textContent 为空时自动拉新闻
- `internal/workflow/nodes/sentiment.go` — news_text 端口文档完善
- `internal/workflow/nodes/stock_research.go` — 接入新闻→情绪
- `internal/workflow/nodes/register.go` — 注册 NewsFetcherNode
- `CHANGELOG.md` — 记录新增

### Phase 2+3: 后续阶段

详见完整 plan (`~/.claude/plans/sequential-orbiting-snowflake.md`)，共 29 个新文件。

## Acceptance Criteria

### Phase 1:
- [ ] `GetSentiment("600519")` 返回基于真实新闻的非 neutral 情绪
- [ ] `SentimentPanel` 显示真实关键词（非 "mock_data"、"frontend_mock"）
- [ ] `NewsFetcherNode → SentimentNode` 工作流端到端运行
- [ ] 新闻适配器单元测试: 600519/688017 返回 ≥3 篇文章
- [ ] 无 Python 桥接时优雅降级到 mock（不 crash）
- [ ] Go 测试全部通过，go vet 无错误

### Phase 2:
- [ ] FinancialsService 返回真实财务数据
- [ ] PeerComparisonService 返回真实行业排名
- [ ] AnalystEstimatesService 返回一致预期 EPS
- [ ] 同花顺热点 + 北向资金 + 概念板块归因可查

### Phase 3:
- [ ] 东财研报 API 可返回研报列表
- [ ] 巨潮公告全文检索可用
- [ ] 同花顺一致预期 EPS 可获取

## Risks / Trade-offs

- **东财封 IP** — 全局限流器 + 开发阶段用缓存优先
- **上游 API 变更** — 每个适配器有独立 `IsAvailable()` 检查，失效时自动跳过
- **新闻文本质量** — 拼接多篇文章增加文本量，提升 NLP 准确度
- **大陆 IP 依赖** — mootdx TCP 和部分东财接口需大陆 IP
- **实现语言**: Go 纯 HTTP（不依赖 Python sidecar），无法复用 SKILL.md 已有 Python 代码但逻辑简单等价
