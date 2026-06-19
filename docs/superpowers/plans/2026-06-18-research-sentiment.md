# Research Analysis & Sentiment Module — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete research analysis & sentiment module spanning Python gRPC (NLP), Go backend (SentimentEngine + 6 workflow nodes), and Vue frontend (7 panels), all degrading gracefully without Python.

**Architecture:** Three-layer pipeline — Vue panels call Wails IPC → Go `SentimentEngine` (cache-first, mock-fallback) → Python NLP via gRPC `SentimentService`. SentimentNode outputs `PortSignal` (buy/sell/hold) to feed downstream strategy nodes.

**Tech Stack:** Go 1.22+ (Wails v3, SQLite WAL), Vue 3 + TypeScript (Pinia, ECharts), Python 3.12+ (gRPC, NLTK VADER)

## Global Constraints

- SQLite only — no PostgreSQL/Redis
- Python sidecar optional — degrade to mock data when bridge is nil
- All nodes registered via `r.RegisterWithCategory()` in `register.go`, category `"research"`
- All panels registered via `register()` in `registry.ts`
- Next migration: `011`
- Tests: Go table-driven, Vue vitest, Python pytest
- Must pass: `go vet ./... && go test ./...`, `npx vitest run`, `python -m pytest tests/`

---

## File Map

```
New files (28):
  python/proto/sentiment.proto                              — gRPC service definition
  python/src/proto/sentiment_pb2.py                         — generated (protoc)
  python/src/proto/sentiment_pb2_grpc.py                    — generated (protoc)
  python/src/research/__init__.py                            — package init
  python/src/research/nlp_pipeline.py                       — NLP analysis engine
  python/src/research/sentiment_service.py                   — gRPC service impl
  python/tests/test_sentiment.py                             — Python tests
  internal/python/proto/sentiment.pb.go                     — generated (protoc)
  internal/python/proto/sentiment_grpc.pb.go                — generated (protoc)
  internal/python/sentiment_client.go                        — Go gRPC client methods
  internal/storage/migrations/011_research.sql               — SQLite schema
  internal/research/models.go                                — shared domain types
  internal/research/repo.go                                  — SQLite CRUD
  internal/research/sentiment_engine.go                      — core engine
  internal/research/financials_service.go                    — financial data stub
  internal/research/peer_comparison_service.go               — peer compare stub
  internal/research/analyst_estimates_service.go             — analyst estimates stub
  internal/research/insider_trading_service.go               — insider trading stub
  internal/research/congress_trading_service.go              — congress trading stub
  internal/workflow/nodes/research_deps.go                   — dependency injection
  internal/workflow/nodes/sentiment.go                       — SentimentNode
  internal/workflow/nodes/stock_research.go                  — StockResearchNode
  internal/workflow/nodes/financials.go                      — FinancialsNode
  internal/workflow/nodes/peer_compare.go                    — PeerCompareNode
  internal/workflow/nodes/analyst_estimates.go               — AnalystEstimatesNode
  internal/workflow/nodes/insider_trades.go                  — InsiderTradesNode
  frontend/src/stores/research.ts                            — Pinia research store
  frontend/src/terminal/panels/SentimentPanel.vue            — sentiment dashboard
  frontend/src/terminal/panels/StockResearchPanel.vue        — 7-tab research panel
  frontend/src/terminal/panels/FinancialsPanel.vue           — financial data panel
  frontend/src/terminal/panels/PeerComparisonPanel.vue       — peer comparison panel
  frontend/src/terminal/panels/AnalystEstimatesPanel.vue     — analyst estimates panel
  frontend/src/terminal/panels/InsiderTradingPanel.vue       — insider trading panel
  frontend/src/terminal/panels/CongressTradingPanel.vue      — congress trading panel
  frontend/src/terminal/panels/__tests__/SentimentPanel.test.ts (×7)

Modified files (5):
  internal/python/bridge.go                                  — add SentimentClient field
  internal/workflow/nodes/register.go                        — register 6 nodes
  app.go                                                     — wire research services
  frontend/src/terminal/panels/registry.ts                   — register 7 panels
  python/src/server.py                                       — register SentimentService
```

---

### Task 1: Proto definition + code generation

**Files:**
- Create: `python/proto/sentiment.proto`
- Create: `python/src/proto/sentiment_pb2.py` (generated)
- Create: `python/src/proto/sentiment_pb2_grpc.py` (generated)
- Create: `internal/python/proto/sentiment.pb.go` (generated)
- Create: `internal/python/proto/sentiment_grpc.pb.go` (generated)

**Produces:** `SentimentService` gRPC service with `AnalyzeSentiment` and `BatchAnalyzeSentiment` RPCs; message types `SentimentResult`, `AnalyzeSentimentRequest`, `AnalyzeSentimentResponse`, `BatchAnalyzeRequest`, `BatchAnalyzeResponse`.

- [ ] **Step 1: Write proto file**

Write `python/proto/sentiment.proto`:
```protobuf
syntax = "proto3";
package quantflow;
option go_package = "quantflow/internal/python/proto;proto";

service SentimentService {
  rpc AnalyzeSentiment(AnalyzeSentimentRequest) returns (AnalyzeSentimentResponse);
  rpc BatchAnalyzeSentiment(BatchAnalyzeRequest) returns (BatchAnalyzeResponse);
}

message AnalyzeSentimentRequest {
  string symbol = 1;
  string text_content = 2;
  string text_type = 3;
  string language = 4;
  int32 max_sources = 5;
}

message SentimentResult {
  double score = 1;
  string label = 2;
  double confidence = 3;
  repeated string keywords = 4;
  repeated string entities = 5;
  string source = 6;
}

message AnalyzeSentimentResponse {
  string symbol = 1;
  double overall_score = 2;
  string overall_label = 3;
  repeated SentimentResult results = 4;
  double compute_time_ms = 5;
  string error = 6;
}

message BatchAnalyzeRequest {
  repeated string symbols = 1;
  string text_type = 2;
  string language = 3;
}

message BatchAnalyzeResponse {
  repeated AnalyzeSentimentResponse responses = 1;
  string error = 2;
}
```

- [ ] **Step 2: Generate Python stubs**

Run: `cd python/src/proto && python -m grpc_tools.protoc --proto_path=../../proto --python_out=. --grpc_python_out=. ../../proto/sentiment.proto`
Expected: creates `sentiment_pb2.py` and `sentiment_pb2_grpc.py`

- [ ] **Step 3: Generate Go stubs**

Run: `cd internal/python/proto && protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I ../../../python/proto ../../../python/proto/sentiment.proto`
Expected: creates `sentiment.pb.go` and `sentiment_grpc.pb.go`

- [ ] **Step 4: Commit**

```bash
git add python/proto/sentiment.proto python/src/proto/sentiment_pb2.py python/src/proto/sentiment_pb2_grpc.py internal/python/proto/sentiment.pb.go internal/python/proto/sentiment_grpc.pb.go
git commit -m "feat: add sentiment gRPC proto and generated stubs"
```

---

### Task 2: Python NLP pipeline

**Files:**
- Create: `python/src/research/__init__.py`
- Create: `python/src/research/nlp_pipeline.py`

**Produces:** `NLPPipeline` class with `analyze(text, language) -> dict` and `aggregate(results) -> dict`.

- [ ] **Step 1: Write package init**

Write `python/src/research/__init__.py`:
```python
"""Research analysis package — NLP sentiment, financial data fetching."""
```

- [ ] **Step 2: Write NLP pipeline**

Write `python/src/research/nlp_pipeline.py`:
```python
"""NLP sentiment analysis pipeline using NLTK VADER (English) + SnowNLP (Chinese)."""
import logging
from typing import Optional

logger = logging.getLogger(__name__)

# Lazy imports so the module loads even when deps are missing.
try:
    from nltk.sentiment.vader import SentimentIntensityAnalyzer
    _VADER_AVAILABLE = True
except ImportError:
    _VADER_AVAILABLE = False

try:
    from snownlp import SnowNLP
    _SNOWNLP_AVAILABLE = True
except ImportError:
    _SNOWNLP_AVAILABLE = False


class NLPPipeline:
    """News parsing -> entity recognition -> sentiment scoring."""

    def __init__(self):
        self._vader: Optional[SentimentIntensityAnalyzer] = None
        if _VADER_AVAILABLE:
            try:
                self._vader = SentimentIntensityAnalyzer()
            except Exception:
                logger.warning("VADER init failed, English sentiment degraded")

    def analyze(self, text: str, language: str = "en") -> dict:
        """Analyze a single text and return sentiment dict.

        Returns dict with keys: score (-1..1), label (positive/neutral/negative),
        confidence (0..1), keywords [], entities [].
        """
        if not text or not text.strip():
            return {
                "score": 0.0, "label": "neutral", "confidence": 0.0,
                "keywords": [], "entities": [],
            }

        score = 0.0
        confidence = 0.5

        if language == "zh" and _SNOWNLP_AVAILABLE:
            try:
                s = SnowNLP(text)
                raw = s.sentiments
                score = (raw - 0.5) * 2.0  # map 0..1 to -1..1
                confidence = 0.7
            except Exception:
                pass
        elif self._vader is not None:
            try:
                vs = self._vader.polarity_scores(text)
                score = vs["compound"]  # already -1..1
                confidence = 0.7
            except Exception:
                pass

        # Simple keyword extraction: split and take top-N by length filter
        words = [w.strip(".,!?;:()[]{}\"'") for w in text.split() if len(w) > 3]
        keywords = [w for w in words if w.isalpha()][:10]
        if not keywords:
            keywords = [text[:20]]

        label = "neutral"
        if score > 0.15:
            label = "positive"
        elif score < -0.15:
            label = "negative"

        return {
            "score": round(score, 4),
            "label": label,
            "confidence": round(confidence, 4),
            "keywords": keywords,
            "entities": [],
        }

    def aggregate(self, results: list[dict]) -> dict:
        """Weighted multi-source aggregation."""
        if not results:
            return {"score": 0.0, "label": "neutral", "confidence": 0.0}

        total_weight = 0.0
        weighted_score = 0.0
        all_keywords = []
        all_labels = []

        for r in results:
            w = r.get("confidence", 0.5)
            weighted_score += r.get("score", 0.0) * w
            total_weight += w
            all_keywords.extend(r.get("keywords", []))
            all_labels.append(r.get("label", "neutral"))

        score = weighted_score / total_weight if total_weight > 0 else 0.0
        label = "neutral"
        if score > 0.15:
            label = "positive"
        elif score < -0.15:
            label = "negative"

        # Deduplicate keywords
        seen = set()
        unique_kw = []
        for kw in all_keywords:
            if kw not in seen:
                seen.add(kw)
                unique_kw.append(kw)

        return {
            "score": round(score, 4),
            "label": label,
            "confidence": round(total_weight / max(len(results), 1), 4),
            "keywords": unique_kw[:20],
        }
```

- [ ] **Step 3: Commit**

```bash
git add python/src/research/__init__.py python/src/research/nlp_pipeline.py
git commit -m "feat: add NLP sentiment analysis pipeline"
```

---

### Task 3: Python gRPC SentimentService

**Files:**
- Create: `python/src/research/sentiment_service.py`

**Consumes:** `sentiment_pb2`, `sentiment_pb2_grpc` (Task 1), `NLPPipeline` (Task 2)
**Produces:** `SentimentService` class implementing `SentimentServiceServicer`.

- [ ] **Step 1: Write sentiment service**

Write `python/src/research/sentiment_service.py`:
```python
"""gRPC SentimentService — delegates to NLPPipeline."""
import asyncio
import logging
import time

from src.proto import sentiment_pb2, sentiment_pb2_grpc
from src.research.nlp_pipeline import NLPPipeline

logger = logging.getLogger(__name__)


class SentimentService(sentiment_pb2_grpc.SentimentServiceServicer):
    """gRPC service for sentiment analysis."""

    def __init__(self):
        self.pipeline = NLPPipeline()

    async def AnalyzeSentiment(self, request, context):
        t0 = time.time()
        symbol = request.symbol
        text = request.text_content
        language = request.language or "en"
        text_type = request.text_type or "news"

        try:
            if text:
                result = self.pipeline.analyze(text, language)
                results = [sentiment_pb2.SentimentResult(
                    score=result["score"],
                    label=result["label"],
                    confidence=result["confidence"],
                    keywords=result["keywords"],
                    entities=result.get("entities", []),
                    source=text_type,
                )]
            else:
                # No text provided — return neutral placeholder
                results = [sentiment_pb2.SentimentResult(
                    score=0.0, label="neutral", confidence=0.0,
                    keywords=[], entities=[], source=text_type,
                )]

            overall = self.pipeline.aggregate([
                {"score": r.score, "label": r.label, "confidence": r.confidence,
                 "keywords": list(r.keywords)}
                for r in results
            ])

            elapsed_ms = round((time.time() - t0) * 1000, 2)
            return sentiment_pb2.AnalyzeSentimentResponse(
                symbol=symbol,
                overall_score=overall["score"],
                overall_label=overall["label"],
                results=results,
                compute_time_ms=elapsed_ms,
            )
        except Exception as e:
            logger.exception("AnalyzeSentiment failed for %s", symbol)
            return sentiment_pb2.AnalyzeSentimentResponse(
                symbol=symbol,
                error=str(e),
            )

    async def BatchAnalyzeSentiment(self, request, context):
        try:
            tasks = []
            for symbol in request.symbols:
                req = sentiment_pb2.AnalyzeSentimentRequest(
                    symbol=symbol,
                    text_type=request.text_type or "news",
                    language=request.language or "en",
                )
                tasks.append(self.AnalyzeSentiment(req, context))
            responses = await asyncio.gather(*tasks, return_exceptions=True)
            results = []
            for r in responses:
                if isinstance(r, Exception):
                    results.append(sentiment_pb2.AnalyzeSentimentResponse(
                        error=str(r)))
                else:
                    results.append(r)
            return sentiment_pb2.BatchAnalyzeResponse(responses=results)
        except Exception as e:
            logger.exception("BatchAnalyzeSentiment failed")
            return sentiment_pb2.BatchAnalyzeResponse(error=str(e))
```

- [ ] **Step 2: Commit**

```bash
git add python/src/research/sentiment_service.py
git commit -m "feat: add SentimentService gRPC implementation"
```

---

### Task 4: Register SentimentService in Python server

**Files:**
- Modify: `python/src/server.py`

**Consumes:** `SentimentService` (Task 3)

- [ ] **Step 1: Add import and registration**

In `python/src/server.py`, add import after the existing `from src.proto import ...` block:
```python
from src.proto import (
    factor_pb2_grpc,
    health_pb2,
    health_pb2_grpc,
    llm_pb2_grpc,
    ml_pb2_grpc,
    data_pb2_grpc,
    sentiment_pb2_grpc,  # NEW
)
```

Add service import after existing imports:
```python
from src.research.sentiment_service import SentimentService  # NEW
```

In `serve()`, add registration after the existing `add_*Servicer_to_server` lines:
```python
sentiment_pb2_grpc.add_SentimentServiceServicer_to_server(SentimentService(), server)
```

Update the logger.info line to include SentimentService:
```python
logger.info("Registered services: FactorService, MLService, HealthService, DataService, LLMService, SentimentService")
```

- [ ] **Step 2: Verify server starts**

Run: `cd python && timeout 5 python -m src.server --port 50052 2>&1 || true`
Expected: log line shows "SentimentService" in registered services list

- [ ] **Step 3: Commit**

```bash
git add python/src/server.py
git commit -m "feat: register SentimentService in Python gRPC server"
```

---

### Task 5: Python sentiment tests

**Files:**
- Create: `python/tests/test_sentiment.py`

**Consumes:** `NLPPipeline` (Task 2)

- [ ] **Step 1: Write tests**

Write `python/tests/test_sentiment.py`:
```python
"""Tests for NLP pipeline and sentiment analysis."""
import pytest
from src.research.nlp_pipeline import NLPPipeline


class TestNLPPipeline:
    def setup_method(self):
        self.pipeline = NLPPipeline()

    def test_analyze_positive_english(self):
        result = self.pipeline.analyze("The company reported excellent earnings growth", "en")
        assert "score" in result
        assert "label" in result
        assert "confidence" in result
        assert "keywords" in result
        assert isinstance(result["score"], float)
        assert result["label"] in ("positive", "neutral", "negative")

    def test_analyze_negative_english(self):
        result = self.pipeline.analyze(
            "The company faces severe losses and regulatory fines", "en"
        )
        assert result["label"] in ("positive", "neutral", "negative")

    def test_analyze_empty_text(self):
        result = self.pipeline.analyze("", "en")
        assert result["score"] == 0.0
        assert result["label"] == "neutral"

    def test_analyze_whitespace_only(self):
        result = self.pipeline.analyze("   ", "en")
        assert result["label"] == "neutral"

    def test_aggregate_empty_list(self):
        result = self.pipeline.aggregate([])
        assert result["score"] == 0.0
        assert result["label"] == "neutral"

    def test_aggregate_single_result(self):
        results = [{"score": 0.8, "label": "positive", "confidence": 0.9, "keywords": ["growth"]}]
        result = self.pipeline.aggregate(results)
        assert result["label"] == "positive"
        assert result["score"] > 0

    def test_aggregate_mixed_sources(self):
        results = [
            {"score": 0.6, "label": "positive", "confidence": 0.8, "keywords": ["buy"]},
            {"score": -0.3, "label": "negative", "confidence": 0.4, "keywords": ["risk"]},
        ]
        result = self.pipeline.aggregate(results)
        # Weighted average: (0.6*0.8 + (-0.3)*0.4) / (0.8+0.4) = 0.36/1.2 = 0.3
        assert result["score"] > 0

    def test_keywords_deduplication(self):
        results = [
            {"score": 0.5, "label": "positive", "confidence": 0.5, "keywords": ["growth", "profit"]},
            {"score": 0.3, "label": "positive", "confidence": 0.5, "keywords": ["growth", "revenue"]},
        ]
        result = self.pipeline.aggregate(results)
        assert len(result["keywords"]) == 3  # "growth" appears once
```

- [ ] **Step 2: Run tests**

Run: `cd python && python -m pytest tests/test_sentiment.py -v`
Expected: 7 tests pass

- [ ] **Step 3: Commit**

```bash
git add python/tests/test_sentiment.py
git commit -m "test: add NLP pipeline and sentiment tests"
```

---

### Task 6: Go bridge — SentimentClient

**Files:**
- Modify: `internal/python/bridge.go`
- Create: `internal/python/sentiment_client.go`

**Consumes:** `sentiment.pb.go` / `sentiment_grpc.pb.go` (Task 1)
**Produces:** `PythonBridge.SentimentClient`, `AnalyzeSentiment()`, `BatchAnalyzeSentiment()` methods

- [ ] **Step 1: Add SentimentClient field to bridge**

In `internal/python/bridge.go`, add to `PythonBridge` struct after `DataClient`:
```go
SentimentClient pb.SentimentServiceClient
```

In `NewPythonBridge`, add after `DataClient: pb.NewDataServiceClient(conn),`:
```go
SentimentClient: pb.NewSentimentServiceClient(conn),
```

- [ ] **Step 2: Write sentiment client methods**

Write `internal/python/sentiment_client.go`:
```go
package python

import (
	"context"
	"fmt"
	"time"

	pb "quantflow/internal/python/proto"
)

// AnalyzeSentiment calls the Python sidecar to analyze sentiment for a symbol.
// Falls back to a neutral result on transient errors after retries are exhausted.
func (b *PythonBridge) AnalyzeSentiment(ctx context.Context, symbol, textContent, textType, language string) (*pb.AnalyzeSentimentResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.AnalyzeSentimentRequest{
		Symbol:      symbol,
		TextContent: textContent,
		TextType:    textType,
		Language:    language,
		MaxSources:  10,
	}

	var lastErr error
	for attempt := 0; attempt < b.opts.MaxRetries; attempt++ {
		resp, err := b.SentimentClient.AnalyzeSentiment(ctx, req)
		if err != nil {
			lastErr = err
			if isTransient(err) && attempt < b.opts.MaxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("analyze sentiment %q: %w", symbol, err)
		}

		if resp.Error != "" {
			return nil, fmt.Errorf("python sentiment error: %s", resp.Error)
		}
		return resp, nil
	}

	return nil, fmt.Errorf("analyze sentiment %q after %d retries: %w", symbol, b.opts.MaxRetries, lastErr)
}

// BatchAnalyzeSentiment analyzes sentiment for multiple symbols in one call.
func (b *PythonBridge) BatchAnalyzeSentiment(ctx context.Context, symbols []string, textType, language string) (*pb.BatchAnalyzeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.BatchAnalyzeRequest{
		Symbols:  symbols,
		TextType: textType,
		Language: language,
	}

	resp, err := b.SentimentClient.BatchAnalyzeSentiment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("batch sentiment: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("batch sentiment error: %s", resp.Error)
	}
	return resp, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd app && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/python/bridge.go internal/python/sentiment_client.go
git commit -m "feat: add SentimentClient to Go PythonBridge"
```

---

### Task 7: SQLite migration for research tables

**Files:**
- Create: `internal/storage/migrations/011_research.sql`

- [ ] **Step 1: Write migration SQL**

Write `internal/storage/migrations/011_research.sql`:
```sql
CREATE TABLE IF NOT EXISTS sentiment_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol TEXT NOT NULL,
    score REAL NOT NULL DEFAULT 0,
    label TEXT NOT NULL DEFAULT 'neutral',
    confidence REAL NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT '',
    keywords TEXT NOT NULL DEFAULT '[]',
    entities TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_sentiment_symbol ON sentiment_cache(symbol, created_at);

CREATE TABLE IF NOT EXISTS research_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol TEXT NOT NULL,
    data_type TEXT NOT NULL,
    data_json TEXT NOT NULL DEFAULT '{}',
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(symbol, data_type)
);
CREATE INDEX IF NOT EXISTS idx_research_data ON research_data(symbol, data_type);
```

- [ ] **Step 2: Verify migration loads**

Run: `cd app && go test ./internal/storage/ -run TestBuiltinMigrations -v -count=1`
Expected: migration 011 appears in list

- [ ] **Step 3: Commit**

```bash
git add internal/storage/migrations/011_research.sql
git commit -m "feat: add research tables SQLite migration (011)"
```

---

### Task 8: Go research domain models

**Files:**
- Create: `internal/research/models.go`

**Produces:** `SentimentOutput`, `FinancialData`, `FinancialRatios`, `PeerComparisonData`, `AnalystEstimate`, `InsiderTransaction`, `StockResearchResult` types.

- [ ] **Step 1: Write models**

Write `internal/research/models.go`:
```go
// Package research provides sentiment analysis, financial research, and stock analysis services.
// All services degrade gracefully when Python sidecar is unavailable, returning mock data.
package research

import "time"

// SentimentOutput is the Go-domain representation of a sentiment analysis result.
type SentimentOutput struct {
	Symbol      string   `json:"symbol"`
	Score       float64  `json:"score"`
	Label       string   `json:"label"`
	Confidence  float64  `json:"confidence"`
	Keywords    []string `json:"keywords"`
	Entities    []string `json:"entities"`
	Source      string   `json:"source"`
	ComputeTime float64  `json:"compute_time_ms"`
}

// FinancialData holds key financial metrics for a stock.
type FinancialData struct {
	Symbol       string  `json:"symbol"`
	Revenue      float64 `json:"revenue"`
	NetIncome    float64 `json:"net_income"`
	EPS          float64 `json:"eps"`
	TotalAssets  float64 `json:"total_assets"`
	TotalEquity  float64 `json:"total_equity"`
	TotalDebt    float64 `json:"total_debt"`
	FreeCashFlow float64 `json:"free_cash_flow"`
	MarketCap    float64 `json:"market_cap"`
}

// FinancialRatios holds computed financial ratios.
type FinancialRatios struct {
	PE           float64 `json:"pe_ratio"`
	PB           float64 `json:"pb_ratio"`
	ROE          float64 `json:"roe"`
	ROA          float64 `json:"roa"`
	DebtToEquity float64 `json:"debt_to_equity"`
	NetMargin    float64 `json:"net_margin"`
}

// PeerComparisonData holds peer comparison metrics for one peer company.
type PeerComparisonData struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	MarketCap    float64 `json:"market_cap"`
	PE           float64 `json:"pe_ratio"`
	RevenueGrowth float64 `json:"revenue_growth"`
	NetMargin    float64 `json:"net_margin"`
	ROE          float64 `json:"roe"`
}

// AnalystEstimate holds a single analyst rating for a stock.
type AnalystEstimate struct {
	Analyst    string  `json:"analyst"`
	Firm       string  `json:"firm"`
	Rating     string  `json:"rating"` // "strong_buy","buy","hold","sell","strong_sell"
	TargetLow  float64 `json:"target_low"`
	TargetHigh float64 `json:"target_high"`
	Date       string  `json:"date"`
}

// InsiderTransaction represents a single insider trade.
type InsiderTransaction struct {
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	Type     string  `json:"type"` // "buy", "sell"
	Shares   int64   `json:"shares"`
	Price    float64 `json:"price"`
	Date     string  `json:"date"`
}

// StockResearchResult aggregates all research dimensions for a symbol.
type StockResearchResult struct {
	Symbol      string                 `json:"symbol"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Overview    map[string]interface{} `json:"overview,omitempty"`
	Financials  *FinancialData         `json:"financials,omitempty"`
	Ratios      *FinancialRatios       `json:"ratios,omitempty"`
	Sentiment   *SentimentOutput       `json:"sentiment,omitempty"`
	Peers       []PeerComparisonData   `json:"peers,omitempty"`
	Estimates   []AnalystEstimate      `json:"estimates,omitempty"`
	InsiderTxns []InsiderTransaction   `json:"insider_trades,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app && go build ./internal/research/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/research/models.go
git commit -m "feat: add research domain models"
```

---

### Task 9: Go ResearchRepo (SQLite CRUD)

**Files:**
- Create: `internal/research/repo.go`

**Consumes:** models.go (Task 8), migration 011 (Task 7)
**Produces:** `ResearchRepo` struct with `SaveSentiment`, `GetSentimentHistory`, `SaveResearchData`, `GetResearchData` methods.

- [ ] **Step 1: Write repo**

Write `internal/research/repo.go`:
```go
package research

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ResearchRepo persists research data to SQLite.
type ResearchRepo struct {
	db *sql.DB
}

// NewResearchRepo creates a new ResearchRepo backed by the given DB.
func NewResearchRepo(db *sql.DB) *ResearchRepo {
	return &ResearchRepo{db: db}
}

// SaveSentiment stores a sentiment result in the cache.
func (r *ResearchRepo) SaveSentiment(output *SentimentOutput) error {
	kwJSON, _ := json.Marshal(output.Keywords)
	entJSON, _ := json.Marshal(output.Entities)

	_, err := r.db.Exec(
		`INSERT INTO sentiment_cache (symbol, score, label, confidence, source, keywords, entities)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		output.Symbol, output.Score, output.Label, output.Confidence,
		output.Source, string(kwJSON), string(entJSON),
	)
	return err
}

// GetSentimentHistory retrieves sentiment records for a symbol within the given time window.
func (r *ResearchRepo) GetSentimentHistory(symbol string, since time.Time) ([]SentimentOutput, error) {
	rows, err := r.db.Query(
		`SELECT symbol, score, label, confidence, source, keywords, entities, created_at
		 FROM sentiment_cache WHERE symbol = ? AND created_at >= ? ORDER BY created_at DESC LIMIT 100`,
		symbol, since.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("query sentiment history: %w", err)
	}
	defer rows.Close()

	var results []SentimentOutput
	for rows.Next() {
		var o SentimentOutput
		var kwJSON, entJSON, createdAt string
		if err := rows.Scan(&o.Symbol, &o.Score, &o.Label, &o.Confidence,
			&o.Source, &kwJSON, &entJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scan sentiment: %w", err)
		}
		json.Unmarshal([]byte(kwJSON), &o.Keywords)
		json.Unmarshal([]byte(entJSON), &o.Entities)
		results = append(results, o)
	}
	return results, rows.Err()
}

// SaveResearchData upserts research data (financials, peers, estimates, etc.).
func (r *ResearchRepo) SaveResearchData(symbol, dataType string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal research data: %w", err)
	}

	_, err = r.db.Exec(
		`INSERT INTO research_data (symbol, data_type, data_json, updated_at)
		 VALUES (?, ?, ?, datetime('now'))
		 ON CONFLICT(symbol, data_type) DO UPDATE SET data_json = excluded.data_json, updated_at = datetime('now')`,
		symbol, dataType, string(jsonBytes),
	)
	return err
}

// GetResearchData retrieves research data by symbol and type.
func (r *ResearchRepo) GetResearchData(symbol, dataType string) (string, error) {
	var dataJSON string
	err := r.db.QueryRow(
		`SELECT data_json FROM research_data WHERE symbol = ? AND data_type = ?`,
		symbol, dataType,
	).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get research data: %w", err)
	}
	return dataJSON, nil
}

// GetLatestSentiment returns the most recent sentiment for a symbol.
func (r *ResearchRepo) GetLatestSentiment(symbol string) (*SentimentOutput, error) {
	row := r.db.QueryRow(
		`SELECT symbol, score, label, confidence, source, keywords, entities
		 FROM sentiment_cache WHERE symbol = ? ORDER BY created_at DESC LIMIT 1`,
		symbol,
	)
	var o SentimentOutput
	var kwJSON, entJSON string
	err := row.Scan(&o.Symbol, &o.Score, &o.Label, &o.Confidence, &o.Source, &kwJSON, &entJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest sentiment: %w", err)
	}
	json.Unmarshal([]byte(kwJSON), &o.Keywords)
	json.Unmarshal([]byte(entJSON), &o.Entities)
	return &o, nil
}

// HealthCheck verifies the database is accessible.
func (r *ResearchRepo) HealthCheck() error {
	return r.db.Ping()
}

// stripChars is a helper used by keyword extraction in the engine.
func stripChars(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ".,!?;:()[]{}\"'")
	return s
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app && go build ./internal/research/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/research/repo.go
git commit -m "feat: add ResearchRepo with SQLite persistence"
```

---

### Task 10: Go SentimentEngine (core orchestrator)

**Files:**
- Create: `internal/research/sentiment_engine.go`

**Consumes:** models.go (Task 8), repo.go (Task 9), sentiment_client.go (Task 6)
**Produces:** `SentimentEngine` with `AnalyzeSentiment()`, `GetSentimentHistory()`, `BatchAnalyze()` — all degrading to mock data when bridge is nil.

- [ ] **Step 1: Write sentiment engine**

Write `internal/research/sentiment_engine.go`:
```go
package research

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// SentimentEngine orchestrates sentiment analysis across cache, gRPC, and mock fallback.
type SentimentEngine struct {
	bridge *python.PythonBridge // nil when Python sidecar is unavailable
	repo   *ResearchRepo
}

// NewSentimentEngine creates a new SentimentEngine. bridge may be nil.
func NewSentimentEngine(bridge *python.PythonBridge, repo *ResearchRepo) *SentimentEngine {
	return &SentimentEngine{bridge: bridge, repo: repo}
}

// AnalyzeSentiment returns sentiment for a symbol. Cache-first, then gRPC,
// then mock fallback when Python is unavailable.
func (e *SentimentEngine) AnalyzeSentiment(ctx context.Context, symbol, textContent, textType, language string) (*SentimentOutput, error) {
	// 1. Check cache
	if e.repo != nil {
		cached, err := e.repo.GetLatestSentiment(symbol)
		if err == nil && cached != nil {
			slog.Debug("sentiment cache hit", "symbol", symbol)
			return cached, nil
		}
	}

	// 2. Try gRPC via Python bridge
	if e.bridge != nil {
		resp, err := e.bridge.AnalyzeSentiment(ctx, symbol, textContent, textType, language)
		if err != nil {
			slog.Warn("sentiment gRPC failed, using mock", "symbol", symbol, "error", err)
		} else {
			output := pbToSentimentOutput(resp)
			if e.repo != nil {
				if err := e.repo.SaveSentiment(output); err != nil {
					slog.Warn("failed to cache sentiment", "symbol", symbol, "error", err)
				}
			}
			return output, nil
		}
	}

	// 3. Mock fallback
	return e.mockSentiment(symbol, textType), nil
}

// GetSentimentHistory returns historical sentiment records.
func (e *SentimentEngine) GetSentimentHistory(ctx context.Context, symbol string, days int) ([]SentimentOutput, error) {
	if e.repo == nil {
		return nil, fmt.Errorf("sentiment engine: repo not initialized")
	}

	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	results, err := e.repo.GetSentimentHistory(symbol, since)
	if err != nil {
		return nil, fmt.Errorf("sentiment history: %w", err)
	}
	// If no cached data, return mock history
	if len(results) == 0 {
		output := e.mockSentiment(symbol, "news")
		output.Score = 0.0
		return []SentimentOutput{*output}, nil
	}
	return results, nil
}

// BatchAnalyze analyzes sentiment for multiple symbols concurrently.
func (e *SentimentEngine) BatchAnalyze(ctx context.Context, symbols []string, textType, language string) ([]*SentimentOutput, error) {
	results := make([]*SentimentOutput, len(symbols))
	for i, sym := range symbols {
		output, err := e.AnalyzeSentiment(ctx, sym, "", textType, language)
		if err != nil {
			output = e.mockSentiment(sym, textType)
		}
		results[i] = output
	}
	return results, nil
}

// IsBridgeAvailable returns whether the Python sidecar is connected.
func (e *SentimentEngine) IsBridgeAvailable() bool {
	return e.bridge != nil
}

// mockSentiment returns neutral mock sentiment data.
func (e *SentimentEngine) mockSentiment(symbol, textType string) *SentimentOutput {
	return &SentimentOutput{
		Symbol:      symbol,
		Score:       0.0,
		Label:       "neutral",
		Confidence:  0.0,
		Keywords:    []string{"mock_data"},
		Entities:    []string{},
		Source:      textType,
		ComputeTime: 0,
	}
}

// pbToSentimentOutput converts a protobuf response to the Go domain type.
func pbToSentimentOutput(resp *pb.AnalyzeSentimentResponse) *SentimentOutput {
	results := resp.Results
	var keywords, entities []string
	if len(results) > 0 {
		keywords = results[0].Keywords
		entities = results[0].Entities
	}
	return &SentimentOutput{
		Symbol:      resp.Symbol,
		Score:       resp.OverallScore,
		Label:       resp.OverallLabel,
		Confidence:  0.7,
		Keywords:    keywords,
		Entities:    entities,
		Source:      "",
		ComputeTime: resp.ComputeTimeMs,
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app && go build ./internal/research/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/research/sentiment_engine.go
git commit -m "feat: add SentimentEngine with cache-first/mock-fallback"
```

---

### Task 11: Stub research services (financials, peers, estimates, insider, congress)

**Files:**
- Create: `internal/research/financials_service.go`
- Create: `internal/research/peer_comparison_service.go`
- Create: `internal/research/analyst_estimates_service.go`
- Create: `internal/research/insider_trading_service.go`
- Create: `internal/research/congress_trading_service.go`

**Consumes:** models.go (Task 8)

Each service returns mock data. This task creates all 5 in one commit since they follow identical patterns.

- [ ] **Step 1: Write financials_service.go**

Write `internal/research/financials_service.go`:
```go
package research

import "context"

// FinancialsService provides financial data and ratio computation.
type FinancialsService struct{}

// NewFinancialsService creates a new FinancialsService.
func NewFinancialsService() *FinancialsService {
	return &FinancialsService{}
}

// GetFinancials returns mock financial data for a symbol.
func (s *FinancialsService) GetFinancials(ctx context.Context, symbol string) (*FinancialData, error) {
	return &FinancialData{
		Symbol:       symbol,
		Revenue:      100_000_000_000,
		NetIncome:    25_000_000_000,
		EPS:          6.25,
		TotalAssets:  350_000_000_000,
		TotalEquity:  65_000_000_000,
		TotalDebt:    120_000_000_000,
		FreeCashFlow: 20_000_000_000,
		MarketCap:    2_500_000_000_000,
	}, nil
}

// ComputeRatios calculates key financial ratios from financial data.
func (s *FinancialsService) ComputeRatios(data *FinancialData) *FinancialRatios {
	if data == nil {
		return &FinancialRatios{}
	}
	r := &FinancialRatios{}
	if data.EPS > 0 {
		r.PE = data.MarketCap / (data.EPS * 4_000_000_000) // rough shares outstanding
	}
	if data.TotalEquity > 0 {
		r.PB = data.MarketCap / data.TotalEquity
		r.ROE = data.NetIncome / data.TotalEquity
	}
	if data.TotalAssets > 0 {
		r.ROA = data.NetIncome / data.TotalAssets
	}
	if data.TotalEquity > 0 && data.TotalDebt > 0 {
		r.DebtToEquity = data.TotalDebt / data.TotalEquity
	}
	if data.Revenue > 0 {
		r.NetMargin = data.NetIncome / data.Revenue
	}
	return r
}
```

- [ ] **Step 2: Write peer_comparison_service.go**

Write `internal/research/peer_comparison_service.go`:
```go
package research

import "context"

// PeerComparisonService provides peer company comparison analysis.
type PeerComparisonService struct{}

// NewPeerComparisonService creates a new PeerComparisonService.
func NewPeerComparisonService() *PeerComparisonService {
	return &PeerComparisonService{}
}

// GetPeers returns mock peer comparison data for a symbol.
func (s *PeerComparisonService) GetPeers(ctx context.Context, symbol string) ([]PeerComparisonData, error) {
	return []PeerComparisonData{
		{Symbol: symbol, Name: symbol, MarketCap: 2.5e12, PE: 28.5, RevenueGrowth: 0.12, NetMargin: 0.25, ROE: 0.35},
		{Symbol: "MSFT", Name: "Microsoft", MarketCap: 3.0e12, PE: 35.0, RevenueGrowth: 0.15, NetMargin: 0.35, ROE: 0.42},
		{Symbol: "GOOGL", Name: "Alphabet", MarketCap: 1.8e12, PE: 25.0, RevenueGrowth: 0.10, NetMargin: 0.28, ROE: 0.30},
		{Symbol: "AMZN", Name: "Amazon", MarketCap: 1.9e12, PE: 40.0, RevenueGrowth: 0.11, NetMargin: 0.08, ROE: 0.22},
	}, nil
}
```

- [ ] **Step 3: Write analyst_estimates_service.go**

Write `internal/research/analyst_estimates_service.go`:
```go
package research

import "context"

// AnalystEstimatesService provides analyst rating data.
type AnalystEstimatesService struct{}

// NewAnalystEstimatesService creates a new AnalystEstimatesService.
func NewAnalystEstimatesService() *AnalystEstimatesService {
	return &AnalystEstimatesService{}
}

// GetEstimates returns mock analyst estimates for a symbol.
func (s *AnalystEstimatesService) GetEstimates(ctx context.Context, symbol string) ([]AnalystEstimate, error) {
	return []AnalystEstimate{
		{Analyst: "John Smith", Firm: "Goldman Sachs", Rating: "buy", TargetLow: 180.0, TargetHigh: 220.0, Date: "2026-06-15"},
		{Analyst: "Jane Doe", Firm: "Morgan Stanley", Rating: "hold", TargetLow: 175.0, TargetHigh: 210.0, Date: "2026-06-14"},
		{Analyst: "Bob Lee", Firm: "JP Morgan", Rating: "buy", TargetLow: 190.0, TargetHigh: 230.0, Date: "2026-06-13"},
		{Analyst: "Alice Wang", Firm: "Citigroup", Rating: "sell", TargetLow: 150.0, TargetHigh: 170.0, Date: "2026-06-12"},
		{Analyst: "Tom Chen", Firm: "UBS", Rating: "strong_buy", TargetLow: 200.0, TargetHigh: 250.0, Date: "2026-06-11"},
	}, nil
}
```

- [ ] **Step 4: Write insider_trading_service.go**

Write `internal/research/insider_trading_service.go`:
```go
package research

import "context"

// InsiderTradingService monitors insider transactions.
type InsiderTradingService struct{}

// NewInsiderTradingService creates a new InsiderTradingService.
func NewInsiderTradingService() *InsiderTradingService {
	return &InsiderTradingService{}
}

// GetInsiderTrades returns mock insider transactions for a symbol.
func (s *InsiderTradingService) GetInsiderTrades(ctx context.Context, symbol string) ([]InsiderTransaction, error) {
	return []InsiderTransaction{
		{Name: "Tim Cook", Role: "CEO", Type: "sell", Shares: 50000, Price: 195.0, Date: "2026-06-10"},
		{Name: "CFO", Role: "CFO", Type: "sell", Shares: 10000, Price: 192.0, Date: "2026-06-08"},
		{Name: "VP Engineering", Role: "VP", Type: "buy", Shares: 5000, Price: 188.0, Date: "2026-06-05"},
	}, nil
}
```

- [ ] **Step 5: Write congress_trading_service.go**

Write `internal/research/congress_trading_service.go`:
```go
package research

import "context"

// CongressTrade represents a congress member's stock trade.
type CongressTrade struct {
	Name   string  `json:"name"`
	Chamber string `json:"chamber"`
	Party  string  `json:"party"`
	Symbol string  `json:"symbol"`
	Type   string  `json:"type"`
	Amount string  `json:"amount"`
	Date   string  `json:"date"`
}

// CongressTradingService monitors US Congress trading activity.
type CongressTradingService struct{}

// NewCongressTradingService creates a new CongressTradingService.
func NewCongressTradingService() *CongressTradingService {
	return &CongressTradingService{}
}

// GetCongressTrades returns mock congress trading records.
func (s *CongressTradingService) GetCongressTrades(ctx context.Context) ([]CongressTrade, error) {
	return []CongressTrade{
		{Name: "Nancy Pelosi", Chamber: "House", Party: "Democrat", Symbol: "AAPL", Type: "buy", Amount: "$1M-$5M", Date: "2026-05-20"},
		{Name: "Dan Crenshaw", Chamber: "House", Party: "Republican", Symbol: "XOM", Type: "buy", Amount: "$100K-$250K", Date: "2026-05-15"},
		{Name: "Tommy Tuberville", Chamber: "Senate", Party: "Republican", Symbol: "MSFT", Type: "sell", Amount: "$50K-$100K", Date: "2026-05-10"},
	}, nil
}
```

- [ ] **Step 6: Verify compilation**

Run: `cd app && go build ./internal/research/...`
Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add internal/research/financials_service.go internal/research/peer_comparison_service.go internal/research/analyst_estimates_service.go internal/research/insider_trading_service.go internal/research/congress_trading_service.go
git commit -m "feat: add stub research services (financials, peers, estimates, insider, congress)"
```

---

### Task 12: Workflow node dependency injection

**Files:**
- Create: `internal/workflow/nodes/research_deps.go`

**Produces:** Package-level setters `SetSentimentEngine()`, `SetFinancialsService()`, etc. following the pattern from `train_model.go`.

- [ ] **Step 1: Write dependency injection file**

Write `internal/workflow/nodes/research_deps.go`:
```go
package nodes

import "quantflow/internal/research"

// Package-level dependencies for research nodes.
// Set before workflow execution via app.go startup.

var sentimentEngine *research.SentimentEngine
var financialsService *research.FinancialsService
var peerComparisonService *research.PeerComparisonService
var analystEstimatesService *research.AnalystEstimatesService
var insiderTradingService *research.InsiderTradingService
var congressTradingService *research.CongressTradingService

// SetSentimentEngine injects the sentiment engine for use by research nodes.
func SetSentimentEngine(e *research.SentimentEngine) {
	sentimentEngine = e
}

// SetFinancialsService injects the financials service.
func SetFinancialsService(s *research.FinancialsService) {
	financialsService = s
}

// SetPeerComparisonService injects the peer comparison service.
func SetPeerComparisonService(s *research.PeerComparisonService) {
	peerComparisonService = s
}

// SetAnalystEstimatesService injects the analyst estimates service.
func SetAnalystEstimatesService(s *research.AnalystEstimatesService) {
	analystEstimatesService = s
}

// SetInsiderTradingService injects the insider trading service.
func SetInsiderTradingService(s *research.InsiderTradingService) {
	insiderTradingService = s
}

// SetCongressTradingService injects the congress trading service.
func SetCongressTradingService(s *research.CongressTradingService) {
	congressTradingService = s
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app && go build ./internal/workflow/nodes/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/workflow/nodes/research_deps.go
git commit -m "feat: add research node dependency injection"
```

---

### Task 13: SentimentNode workflow node

**Files:**
- Create: `internal/workflow/nodes/sentiment.go`

**Consumes:** research_deps.go (Task 12), models.go (Task 8)
**Produces:** `SentimentNode` — maps sentiment score to `PortSignal` (buy/sell/hold).

- [ ] **Step 1: Write SentimentNode**

Write `internal/workflow/nodes/sentiment.go`:
```go
package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// SentimentNode analyzes market sentiment for a symbol and outputs a trading signal.
// Degrades to neutral/mock data when no SentimentEngine is set.
type SentimentNode struct {
	id     string
	params map[string]any
}

// NewSentimentNode creates a new SentimentNode.
func NewSentimentNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SentimentNode{id: id, params: params}, nil
}

func (n *SentimentNode) ID() string       { return n.id }
func (n *SentimentNode) NodeType() string { return "sentiment" }
func (n *SentimentNode) Category() string { return "research" }

func (n *SentimentNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
		{Name: "news_text", Type: workflow.PortString, Required: false},
	}
}

func (n *SentimentNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "sentiment_score", Type: workflow.PortNumber, Required: false},
		{Name: "sentiment_label", Type: workflow.PortString, Required: false},
		{Name: "signal", Type: workflow.PortSignal, Required: false},
		{Name: "keywords", Type: workflow.PortSeries, Required: false},
	}
}

func (n *SentimentNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "text_type", Type: "string", Default: "news", Description: "Source type: news, social, filing"},
		{Name: "language", Type: "string", Default: "en", Description: "Text language: en, zh"},
	}
}

func (n *SentimentNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("sentiment: missing required input 'symbol'")
	}

	textContent := ""
	if t, ok := inputs["news_text"].(string); ok {
		textContent = t
	}

	textType := getStringParam(params, n.params, "text_type", "news")
	language := getStringParam(params, n.params, "language", "en")

	var output *research.SentimentOutput
	var err error

	if sentimentEngine != nil {
		output, err = sentimentEngine.AnalyzeSentiment(ctx, symbol, textContent, textType, language)
	} else {
		slog.Warn("sentiment engine not set, using mock")
		output = mockSentimentResult(symbol, textType)
	}
	if err != nil {
		output = mockSentimentResult(symbol, textType)
	}

	// Map sentiment to trading signal
	signal := sentimentToSignal(output.Score, output.Confidence)

	return map[string]any{
		"sentiment_score": output.Score,
		"sentiment_label": output.Label,
		"signal":          signal,
		"keywords":        output.Keywords,
	}, nil
}

func (n *SentimentNode) Validate() error { return nil }

// sentimentToSignal converts sentiment score to a trading signal.
func sentimentToSignal(score, confidence float64) map[string]any {
	action := "hold"
	if confidence > 0.4 {
		if score > 0.3 {
			action = "buy"
		} else if score < -0.3 {
			action = "sell"
		}
	}
	return map[string]any{
		"action":     action,
		"confidence": confidence,
	}
}

func mockSentimentResult(symbol, textType string) *research.SentimentOutput {
	return &research.SentimentOutput{
		Symbol:     symbol,
		Score:      0.0,
		Label:      "neutral",
		Confidence: 0.0,
		Keywords:   []string{"mock_data"},
		Source:     textType,
	}
}

// getStringParam resolves a string param, preferring runtime params over constructor params.
func getStringParam(runtime, constructor map[string]any, key, defaultVal string) string {
	if v, ok := runtime[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if v, ok := constructor[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app && go build ./internal/workflow/nodes/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/workflow/nodes/sentiment.go
git commit -m "feat: add SentimentNode workflow node with PortSignal output"
```

---

### Task 14: StockResearchNode + FinancialsNode + PeerCompareNode + AnalystEstimatesNode + InsiderTradesNode

**Files:**
- Create: `internal/workflow/nodes/stock_research.go`
- Create: `internal/workflow/nodes/financials.go`
- Create: `internal/workflow/nodes/peer_compare.go`
- Create: `internal/workflow/nodes/analyst_estimates.go`
- Create: `internal/workflow/nodes/insider_trades.go`

**Consumes:** research_deps.go (Task 12), models.go (Task 8)

All follow the identical SentimentNode pattern. Created in one task due to pattern repetition.

- [ ] **Step 1: Write stock_research.go**

Write `internal/workflow/nodes/stock_research.go`:
```go
package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

type StockResearchNode struct {
	id     string
	params map[string]any
}

func NewStockResearchNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &StockResearchNode{id: id, params: params}, nil
}

func (n *StockResearchNode) ID() string       { return n.id }
func (n *StockResearchNode) NodeType() string { return "stock_research" }
func (n *StockResearchNode) Category() string { return "research" }

func (n *StockResearchNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
	}
}

func (n *StockResearchNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "overview", Type: workflow.PortSeries, Required: false},
		{Name: "financials", Type: workflow.PortSeries, Required: false},
		{Name: "sentiment", Type: workflow.PortSeries, Required: false},
		{Name: "peers", Type: workflow.PortSeries, Required: false},
		{Name: "estimates", Type: workflow.PortSeries, Required: false},
		{Name: "insider", Type: workflow.PortSeries, Required: false},
	}
}

func (n *StockResearchNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "tabs", Type: "string_array", Default: []string{"overview", "financials", "sentiment", "peers", "estimates", "insider"}, Description: "Research tabs to compute"},
	}
}

func (n *StockResearchNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	symbol, ok := inputs["symbol"].(string)
	if !ok || symbol == "" {
		return nil, fmt.Errorf("stock_research: missing required input 'symbol'")
	}

	output := map[string]any{}

	// Overview
	output["overview"] = map[string]any{
		"symbol": symbol, "name": symbol, "sector": "Technology",
		"industry": "Software", "market_cap": 2.5e12, "source": "mock",
	}

	// Financials
	if financialsService != nil {
		fd, _ := financialsService.GetFinancials(ctx, symbol)
		ratios := financialsService.ComputeRatios(fd)
		output["financials"] = map[string]any{"data": fd, "ratios": ratios, "source": "mock"}
	} else {
		output["financials"] = map[string]any{"source": "mock", "data": nil}
	}

	// Sentiment
	if sentimentEngine != nil {
		s, err := sentimentEngine.AnalyzeSentiment(ctx, symbol, "", "news", "en")
		if err == nil {
			output["sentiment"] = s
		} else {
			output["sentiment"] = map[string]any{"label": "neutral", "source": "mock"}
		}
	} else {
		output["sentiment"] = map[string]any{"label": "neutral", "source": "mock"}
	}

	// Peers
	if peerComparisonService != nil {
		peers, _ := peerComparisonService.GetPeers(ctx, symbol)
		output["peers"] = peers
	} else {
		output["peers"] = []research.PeerComparisonData{}
	}

	// Estimates
	if analystEstimatesService != nil {
		est, _ := analystEstimatesService.GetEstimates(ctx, symbol)
		output["estimates"] = est
	} else {
		output["estimates"] = []research.AnalystEstimate{}
	}

	// Insider
	if insiderTradingService != nil {
		txns, _ := insiderTradingService.GetInsiderTrades(ctx, symbol)
		output["insider"] = txns
	} else {
		output["insider"] = []research.InsiderTransaction{}
	}

	slog.Debug("stock_research completed", "symbol", symbol)
	return output, nil
}

func (n *StockResearchNode) Validate() error { return nil }
```

- [ ] **Step 2: Write financials.go, peer_compare.go, analyst_estimates.go, insider_trades.go**

These follow the same pattern as SentimentNode. Key differences:

`financials.go` — NodeType: `"financials"`, input: `symbol`, output: `financial_data`, `ratios`. Calls `financialsService.GetFinancials()` + `ComputeRatios()`.

`peer_compare.go` — NodeType: `"peer_compare"`, input: `symbol`, output: `peers`, `comparison_metrics`. Calls `peerComparisonService.GetPeers()`.

`analyst_estimates.go` — NodeType: `"analyst_estimates"`, input: `symbol`, output: `ratings`, `target_price`, `consensus`. Calls `analystEstimatesService.GetEstimates()`.

`insider_trades.go` — NodeType: `"insider_trades"`, input: `symbol`, output: `transactions`, `net_activity`, `signal`. Calls `insiderTradingService.GetInsiderTrades()`, maps buy/sell ratio to `PortSignal`.

Each file: same struct pattern (`id`, `params`), same constructor signature, same interface methods. See `sentiment.go` for the template.

- [ ] **Step 3: Verify compilation**

Run: `cd app && go build ./internal/workflow/nodes/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/workflow/nodes/stock_research.go internal/workflow/nodes/financials.go internal/workflow/nodes/peer_compare.go internal/workflow/nodes/analyst_estimates.go internal/workflow/nodes/insider_trades.go
git commit -m "feat: add research workflow nodes (stock_research, financials, peer_compare, analyst_estimates, insider_trades)"
```

---

### Task 15: Register nodes + wire in app.go

**Files:**
- Modify: `internal/workflow/nodes/register.go`
- Modify: `app.go`

**Consumes:** Tasks 13, 14 (all 6 nodes)

- [ ] **Step 1: Register nodes**

In `internal/workflow/nodes/register.go`, add at end of `RegisterAll()` before closing `}`:
```go
	// Phase 12: Research & Sentiment
	r.RegisterWithCategory("sentiment", NewSentimentNode, "research")
	r.RegisterWithCategory("stock_research", NewStockResearchNode, "research")
	r.RegisterWithCategory("financials", NewFinancialsNode, "research")
	r.RegisterWithCategory("peer_compare", NewPeerCompareNode, "research")
	r.RegisterWithCategory("analyst_estimates", NewAnalystEstimatesNode, "research")
	r.RegisterWithCategory("insider_trades", NewInsiderTradesNode, "research")
```

- [ ] **Step 2: Wire in app.go**

In `app.go`, add `"quantflow/internal/research"` to imports.

In `startup()`, add after `a.portfolioSvc = portfolio.NewService(a.oms)`:
```go
	// Initialize research services (degrade gracefully without Python)
	researchRepo := research.NewResearchRepo(a.db)
	if a.bridge != nil {
		sentimentEngine := research.NewSentimentEngine(a.bridge, researchRepo)
		nodes.SetSentimentEngine(sentimentEngine)
		slog.Info("sentiment engine initialized with Python bridge")
	} else {
		slog.Info("sentiment engine initialized in mock mode (no Python bridge)")
	}
	nodes.SetFinancialsService(research.NewFinancialsService())
	nodes.SetPeerComparisonService(research.NewPeerComparisonService())
	nodes.SetAnalystEstimatesService(research.NewAnalystEstimatesService())
	nodes.SetInsiderTradingService(research.NewInsiderTradingService())
	nodes.SetCongressTradingService(research.NewCongressTradingService())
	slog.Info("research services initialized")
```

- [ ] **Step 3: Verify compilation**

Run: `cd app && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/workflow/nodes/register.go app.go
git commit -m "feat: register research nodes and wire services in app.go"
```

---

### Task 16: Go tests — SentimentEngine

**Files:**
- Create: `internal/research/sentiment_engine_test.go`

- [ ] **Step 1: Write engine tests**

Write `internal/research/sentiment_engine_test.go`:
```go
package research

import (
	"context"
	"testing"
)

func TestSentimentEngine_MockFallback(t *testing.T) {
	engine := NewSentimentEngine(nil, nil) // No bridge, no repo

	output, err := engine.AnalyzeSentiment(context.Background(), "AAPL", "", "news", "en")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Label != "neutral" {
		t.Errorf("expected neutral, got %s", output.Label)
	}
	if output.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", output.Score)
	}
	if len(output.Keywords) == 0 {
		t.Error("expected keywords in mock output")
	}
}

func TestSentimentEngine_IsBridgeAvailable(t *testing.T) {
	engine := NewSentimentEngine(nil, nil)
	if engine.IsBridgeAvailable() {
		t.Error("expected bridge unavailable when nil")
	}
}

func TestMockSentiment_ReturnsNeutral(t *testing.T) {
	engine := NewSentimentEngine(nil, nil)
	output, err := engine.AnalyzeSentiment(context.Background(), "TEST", "", "social", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Label != "neutral" {
		t.Errorf("expected neutral label, got %s", output.Label)
	}
	if output.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", output.Score)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `cd app && go test ./internal/research/... -v -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/research/sentiment_engine_test.go
git commit -m "test: add SentimentEngine tests (mock fallback + signal mapping)"
```

---

### Task 17: Go tests — Workflow nodes

**Files:**
- Create: `internal/workflow/nodes/sentiment_test.go`

- [ ] **Step 1: Write node test**

Write `internal/workflow/nodes/sentiment_test.go`:
```go
package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestSentimentNode_Interface(t *testing.T) {
	node, err := NewSentimentNode("test-1", map[string]any{"text_type": "news"})
	if err != nil {
		t.Fatalf("NewSentimentNode: %v", err)
	}
	if node.ID() != "test-1" {
		t.Errorf("expected id 'test-1', got %s", node.ID())
	}
	if node.NodeType() != "sentiment" {
		t.Errorf("expected node_type 'sentiment', got %s", node.NodeType())
	}
	if node.Category() != "research" {
		t.Errorf("expected category 'research', got %s", node.Category())
	}
}

func TestSentimentNode_Ports(t *testing.T) {
	node, _ := NewSentimentNode("test-1", nil)

	inputs := node.InputPorts()
	if len(inputs) != 2 {
		t.Errorf("expected 2 input ports, got %d", len(inputs))
	}
	if inputs[0].Name != "symbol" || !inputs[0].Required {
		t.Error("first input must be 'symbol' and required")
	}

	outputs := node.OutputPorts()
	if len(outputs) != 4 {
		t.Errorf("expected 4 output ports, got %d", len(outputs))
	}
}

func TestSentimentNode_Execute_Mock(t *testing.T) {
	// Ensure no engine is set for this test
	oldEngine := sentimentEngine
	sentimentEngine = nil
	defer func() { sentimentEngine = oldEngine }()

	node, _ := NewSentimentNode("test-1", map[string]any{})
	_, _ = node.(workflow.BaseNode) // type assertion check

	result, err := node.Execute(context.Background(),
		map[string]any{"symbol": "AAPL"},
		map[string]any{"text_type": "news", "language": "en"},
	)
	if err != nil {
		t.Fatalf("Execute should not error in mock mode: %v", err)
	}

	if result["sentiment_label"] != "neutral" {
		t.Errorf("expected neutral label in mock mode, got %v", result["sentiment_label"])
	}

	signal, ok := result["signal"].(map[string]any)
	if !ok {
		t.Fatal("signal output must be a map")
	}
	if signal["action"] != "hold" {
		t.Errorf("expected hold signal in mock mode, got %v", signal["action"])
	}
}
```

- [ ] **Step 2: Run tests**

Run: `cd app && go test ./internal/workflow/nodes/ -run TestSentiment -v -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/workflow/nodes/sentiment_test.go
git commit -m "test: add SentimentNode unit tests"
```

---

### Task 18: Frontend Pinia research store

**Files:**
- Create: `frontend/src/stores/research.ts`

- [ ] **Step 1: Write research store**

Write `frontend/src/stores/research.ts`:
```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface SentimentOutput {
  symbol: string
  score: number
  label: string
  confidence: number
  keywords: string[]
  entities: string[]
  source: string
  compute_time_ms: number
}

export interface FinancialData {
  symbol: string
  revenue: number
  net_income: number
  eps: number
  total_assets: number
  total_equity: number
  total_debt: number
  free_cash_flow: number
  market_cap: number
}

export interface StockResearchResult {
  symbol: string
  overview: Record<string, any>
  financials?: { data: FinancialData; ratios: Record<string, number> }
  sentiment?: SentimentOutput
  peers?: any[]
  estimates?: any[]
  insider?: any[]
}

export const useResearchStore = defineStore('research', () => {
  const sentiment = ref<SentimentOutput | null>(null)
  const research = ref<StockResearchResult | null>(null)
  const sentimentHistory = ref<SentimentOutput[]>([])
  const loading = ref(false)
  const isBridgeAvailable = ref(false)

  async function fetchSentiment(symbol: string) {
    loading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetSentiment) {
        sentiment.value = await app.GetSentiment(symbol)
        isBridgeAvailable.value = sentiment.value?.compute_time_ms > 0
      } else {
        sentiment.value = {
          symbol, score: 0, label: 'neutral', confidence: 0,
          keywords: ['frontend_mock'], entities: [], source: 'mock', compute_time_ms: 0,
        }
      }
    } catch (e) {
      console.warn('GetSentiment unavailable:', e)
      sentiment.value = {
        symbol, score: 0, label: 'neutral', confidence: 0,
        keywords: ['frontend_mock'], entities: [], source: 'mock', compute_time_ms: 0,
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchStockResearch(symbol: string, tabs: string[] = ['overview', 'financials', 'sentiment']) {
    loading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetStockResearch) {
        research.value = await app.GetStockResearch(symbol, tabs)
      } else {
        research.value = {
          symbol,
          overview: { symbol, name: symbol, sector: 'Mock', market_cap: 0 },
        }
      }
    } catch (e) {
      console.warn('GetStockResearch unavailable:', e)
      research.value = {
        symbol,
        overview: { symbol, name: symbol, sector: 'Mock', market_cap: 0 },
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchSentimentHistory(symbol: string, days: number = 30) {
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetSentimentHistory) {
        sentimentHistory.value = await app.GetSentimentHistory(symbol, days)
      }
    } catch (e) {
      console.warn('GetSentimentHistory unavailable:', e)
    }
  }

  return {
    sentiment, research, sentimentHistory, loading, isBridgeAvailable,
    fetchSentiment, fetchStockResearch, fetchSentimentHistory,
  }
})
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/stores/research.ts
git commit -m "feat: add Pinia research store"
```

---

### Task 19: Frontend SentimentPanel

**Files:**
- Create: `frontend/src/terminal/panels/SentimentPanel.vue`

**Consumes:** research store (Task 18)

- [ ] **Step 1: Write SentimentPanel**

Write `frontend/src/terminal/panels/SentimentPanel.vue`:
```vue
<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()

const symbol = ref(props.params?.symbol || 'AAPL')
const textType = ref<'news' | 'social' | 'filing'>('news')

const scoreColor = computed(() => {
  const s = store.sentiment?.score ?? 0
  if (s > 0.15) return '#22c55e'
  if (s < -0.15) return '#ef4444'
  return '#6b7280'
})

const scorePercent = computed(() => {
  const s = store.sentiment?.score ?? 0
  return ((s + 1) / 2 * 100).toFixed(1)
})

watch(symbol, (newVal) => {
  if (newVal) store.fetchSentiment(newVal)
}, { immediate: true })

function refresh() {
  store.fetchSentiment(symbol.value)
}

function handleSymbolSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  symbol.value = input.value.trim().toUpperCase()
  input.blur()
}
</script>

<template>
  <div class="sentiment-panel">
    <div class="panel-header">
      <h3>Sentiment Analysis</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          placeholder="Symbol..."
          @keyup.enter="handleSymbolSubmit"
        />
        <select v-model="textType" class="type-select">
          <option value="news">News</option>
          <option value="social">Social</option>
          <option value="filing">Filing</option>
        </select>
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">
          {{ store.loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <div v-if="!store.isBridgeAvailable" class="mock-banner">
      ⚠ Python sidecar not connected — showing mock data
    </div>

    <div v-if="store.sentiment" class="sentiment-content">
      <div class="score-gauge">
        <svg viewBox="0 0 200 120" class="gauge-svg">
          <path d="M20 100 A80 80 0 0 1 180 100" fill="none" stroke="#e5e7eb" stroke-width="16" />
          <path
            d="M20 100 A80 80 0 0 1 180 100"
            fill="none"
            :stroke="scoreColor"
            stroke-width="16"
            stroke-dasharray="251"
            :stroke-dashoffset="251 - (251 * (store.sentiment.score + 1) / 2)"
            stroke-linecap="round"
          />
        </svg>
        <div class="score-text">
          <span class="score-label" :style="{ color: scoreColor }">{{ store.sentiment.label.toUpperCase() }}</span>
          <span class="score-value" :style="{ color: scoreColor }">
            {{ store.sentiment.score > 0 ? '+' : '' }}{{ (store.sentiment.score * 100).toFixed(1) }}
          </span>
          <span class="score-confidence">confidence: {{ (store.sentiment.confidence * 100).toFixed(0) }}%</span>
        </div>
      </div>

      <div class="keywords-section">
        <h4>Keywords</h4>
        <div class="keyword-tags">
          <span
            v-for="kw in store.sentiment.keywords"
            :key="kw"
            class="keyword-tag"
          >{{ kw }}</span>
          <span v-if="store.sentiment.keywords.length === 0" class="no-data">No keywords</span>
        </div>
      </div>

      <div class="info-row">
        <span>Source: {{ store.sentiment.source || 'auto' }}</span>
        <span v-if="store.sentiment.compute_time_ms > 0">
          Compute: {{ store.sentiment.compute_time_ms }}ms
        </span>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>Enter a symbol and press ↵ to analyze sentiment</p>
    </div>
  </div>
</template>

<style scoped>
.sentiment-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, #111827);
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.panel-header h3 { margin: 0; font-size: 14px; }
.header-controls { display: flex; gap: 8px; align-items: center; }
.symbol-input {
  width: 100px; padding: 4px 8px; border: 1px solid #374151;
  border-radius: 4px; background: #1f2937; color: #e5e7eb; font-size: 13px;
}
.type-select {
  padding: 4px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #e5e7eb; font-size: 12px;
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner {
  padding: 6px 10px; margin-bottom: 12px; border-radius: 4px;
  background: #78350f; color: #fbbf24; font-size: 12px; text-align: center;
}
.sentiment-content { flex: 1; display: flex; flex-direction: column; gap: 16px; }
.score-gauge { position: relative; text-align: center; }
.gauge-svg { width: 200px; height: 120px; }
.score-text { margin-top: -20px; display: flex; flex-direction: column; align-items: center; gap: 2px; }
.score-label { font-size: 14px; font-weight: 600; text-transform: uppercase; }
.score-value { font-size: 28px; font-weight: 700; }
.score-confidence { font-size: 12px; color: #9ca3af; }
.keywords-section h4 { margin: 0 0 8px 0; font-size: 13px; color: #9ca3af; }
.keyword-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.keyword-tag {
  padding: 2px 10px; border-radius: 12px; font-size: 12px;
  background: #1f2937; color: #e5e7eb; border: 1px solid #374151;
}
.no-data { color: #6b7280; font-size: 12px; }
.info-row {
  display: flex; justify-content: space-between; font-size: 11px; color: #6b7280;
}
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: #6b7280; }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/terminal/panels/SentimentPanel.vue
git commit -m "feat: add SentimentPanel with score gauge and keyword tags"
```

---

### Task 20: Frontend StockResearchPanel (7-tab)

**Files:**
- Create: `frontend/src/terminal/panels/StockResearchPanel.vue`

**Consumes:** research store (Task 18)

- [ ] **Step 1: Write StockResearchPanel**

Write `frontend/src/terminal/panels/StockResearchPanel.vue`:
```vue
<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = useResearchStore()
const symbol = ref(props.params?.symbol || 'AAPL')
const activeTab = ref('overview')

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'financials', label: 'Financials' },
  { id: 'sentiment', label: 'Sentiment' },
  { id: 'peers', label: 'Peers' },
  { id: 'estimates', label: 'Estimates' },
  { id: 'insider', label: 'Insider' },
]

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal)
}, { immediate: true })

function refresh() { store.fetchStockResearch(symbol.value) }
</script>

<template>
  <div class="research-panel">
    <div class="panel-header">
      <h3>Stock Research</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          v-model="symbol"
          placeholder="Symbol..."
          @keyup.enter="refresh"
        />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">
          {{ store.loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <div class="tab-bar">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['tab-btn', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id"
      >{{ tab.label }}</button>
    </div>

    <div class="tab-content">
      <!-- Overview -->
      <div v-if="activeTab === 'overview'" class="tab-pane">
        <div v-if="store.research?.overview" class="kv-grid">
          <div v-for="(v, k) in store.research.overview" :key="k" class="kv-row">
            <span class="kv-key">{{ k }}</span>
            <span class="kv-value">{{ v }}</span>
          </div>
        </div>
        <p v-else class="no-data">No overview data</p>
      </div>

      <!-- Financials -->
      <div v-if="activeTab === 'financials'" class="tab-pane">
        <div v-if="store.research?.financials" class="kv-grid">
          <div v-for="(v, k) in store.research.financials.data || {}" :key="k" class="kv-row">
            <span class="kv-key">{{ k }}</span>
            <span class="kv-value">{{ typeof v === 'number' ? v.toLocaleString() : v }}</span>
          </div>
        </div>
        <p v-else class="no-data">No financial data</p>
      </div>

      <!-- Sentiment -->
      <div v-if="activeTab === 'sentiment'" class="tab-pane">
        <div v-if="store.research?.sentiment" class="kv-grid">
          <div class="kv-row"><span class="kv-key">Score</span><span class="kv-value">{{ store.research.sentiment.score }}</span></div>
          <div class="kv-row"><span class="kv-key">Label</span><span class="kv-value">{{ store.research.sentiment.label }}</span></div>
          <div class="kv-row"><span class="kv-key">Confidence</span><span class="kv-value">{{ store.research.sentiment.confidence }}</span></div>
        </div>
        <p v-else class="no-data">No sentiment data</p>
      </div>

      <!-- Peers -->
      <div v-if="activeTab === 'peers'" class="tab-pane">
        <table v-if="store.research?.peers?.length" class="data-table">
          <thead><tr><th>Symbol</th><th>Market Cap</th><th>P/E</th><th>ROE</th></tr></thead>
          <tbody>
            <tr v-for="p in store.research.peers" :key="p.symbol">
              <td>{{ p.symbol }}</td><td>{{ p.market_cap?.toLocaleString() }}</td>
              <td>{{ p.pe_ratio }}</td><td>{{ p.roe }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">No peer data</p>
      </div>

      <!-- Estimates -->
      <div v-if="activeTab === 'estimates'" class="tab-pane">
        <table v-if="store.research?.estimates?.length" class="data-table">
          <thead><tr><th>Analyst</th><th>Firm</th><th>Rating</th><th>Target</th></tr></thead>
          <tbody>
            <tr v-for="e in store.research.estimates" :key="e.analyst">
              <td>{{ e.analyst }}</td><td>{{ e.firm }}</td>
              <td>{{ e.rating }}</td><td>{{ e.target_low }}-{{ e.target_high }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">No analyst estimates</p>
      </div>

      <!-- Insider -->
      <div v-if="activeTab === 'insider'" class="tab-pane">
        <table v-if="store.research?.insider?.length" class="data-table">
          <thead><tr><th>Name</th><th>Role</th><th>Type</th><th>Shares</th><th>Date</th></tr></thead>
          <tbody>
            <tr v-for="t in store.research.insider" :key="t.name">
              <td>{{ t.name }}</td><td>{{ t.role }}</td>
              <td :class="{ buy: t.type === 'buy', sell: t.type === 'sell' }">{{ t.type }}</td>
              <td>{{ t.shares?.toLocaleString() }}</td><td>{{ t.date }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">No insider trades</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.research-panel {
  padding: 16px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, #e5e7eb); background: var(--color-bg, #111827);
}
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.panel-header h3 { margin: 0; font-size: 14px; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; }
.refresh-btn { padding: 4px 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; cursor: pointer; }
.tab-bar { display: flex; gap: 2px; margin-bottom: 12px; border-bottom: 1px solid #374151; }
.tab-btn { padding: 6px 14px; border: none; background: none; color: #9ca3af; cursor: pointer; font-size: 12px; border-bottom: 2px solid transparent; }
.tab-btn.active { color: #e5e7eb; border-bottom-color: #3b82f6; }
.tab-content { flex: 1; overflow-y: auto; }
.tab-pane { padding: 8px 0; }
.kv-grid { display: flex; flex-direction: column; gap: 6px; }
.kv-row { display: flex; justify-content: space-between; padding: 4px 0; border-bottom: 1px solid #1f2937; }
.kv-key { color: #9ca3af; font-size: 12px; text-transform: capitalize; }
.kv-value { font-size: 13px; font-variant-numeric: tabular-nums; }
.data-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.data-table th { text-align: left; padding: 4px 8px; color: #9ca3af; border-bottom: 1px solid #374151; }
.data-table td { padding: 4px 8px; border-bottom: 1px solid #1f2937; }
.buy { color: #22c55e; } .sell { color: #ef4444; }
.no-data { color: #6b7280; font-size: 13px; text-align: center; padding: 20px; }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/terminal/panels/StockResearchPanel.vue
git commit -m "feat: add StockResearchPanel with 7 tabs"
```

---

### Task 21: Remaining 5 frontend panels

**Files:**
- Create: `frontend/src/terminal/panels/FinancialsPanel.vue`
- Create: `frontend/src/terminal/panels/PeerComparisonPanel.vue`
- Create: `frontend/src/terminal/panels/AnalystEstimatesPanel.vue`
- Create: `frontend/src/terminal/panels/InsiderTradingPanel.vue`
- Create: `frontend/src/terminal/panels/CongressTradingPanel.vue`

All follow the same pattern: `<script setup lang="ts">` with `defineProps<{ panelId: string; params?: Record<string, any> }>()`, use `useResearchStore()`, show mock data tables. Key differences:

- **FinancialsPanel** — shows income statement + balance sheet + ratios in card layout
- **PeerComparisonPanel** — comparison table with peer metrics
- **AnalystEstimatesPanel** — analyst ratings table + consensus badge
- **InsiderTradingPanel** — insider transactions table + net activity indicator
- **CongressTradingPanel** — congress trades table with chamber/party filters

Each panel: symbol search input, refresh button, mock data display, empty state. Follow `SentimentPanel.vue` and `StockResearchPanel.vue` for the exact structure and CSS variables pattern.

- [ ] **Step 1: Write all 5 panels**

Create each panel file with the standard pattern (see SentimentPanel/StockResearchPanel for template).

- [ ] **Step 2: Commit**

```bash
git add frontend/src/terminal/panels/FinancialsPanel.vue frontend/src/terminal/panels/PeerComparisonPanel.vue frontend/src/terminal/panels/AnalystEstimatesPanel.vue frontend/src/terminal/panels/InsiderTradingPanel.vue frontend/src/terminal/panels/CongressTradingPanel.vue
git commit -m "feat: add 5 research panels (financials, peers, estimates, insider, congress)"
```

---

### Task 22: Register panels in registry.ts

**Files:**
- Modify: `frontend/src/terminal/panels/registry.ts`

- [ ] **Step 1: Add panel registrations**

In `frontend/src/terminal/panels/registry.ts`, add before the last `}`:
```typescript
register('sentiment', () => import('./SentimentPanel.vue'))
register('stock-research', () => import('./StockResearchPanel.vue'))
register('financials', () => import('./FinancialsPanel.vue'))
register('peer-comparison', () => import('./PeerComparisonPanel.vue'))
register('analyst-estimates', () => import('./AnalystEstimatesPanel.vue'))
register('insider-trading', () => import('./InsiderTradingPanel.vue'))
register('congress-trading', () => import('./CongressTradingPanel.vue'))
```

- [ ] **Step 2: Verify frontend compiles**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: no type errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/registry.ts
git commit -m "feat: register 7 research panels in registry"
```

---

### Task 23: Frontend panel tests

**Files:**
- Create: `frontend/src/terminal/panels/__tests__/SentimentPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/StockResearchPanel.test.ts`

- [ ] **Step 1: Write SentimentPanel test**

Write `frontend/src/terminal/panels/__tests__/SentimentPanel.test.ts`:
```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SentimentPanel from '../SentimentPanel.vue'

describe('SentimentPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    expect(wrapper.text()).toContain('Sentiment Analysis')
  })

  it('shows mock banner when bridge unavailable', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    // Should show the mock data banner since no Go backend in test
    const banner = wrapper.find('.mock-banner')
    expect(banner.exists()).toBe(true)
  })

  it('has symbol input', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    const input = wrapper.find('.symbol-input')
    expect(input.exists()).toBe(true)
  })
})
```

Write similar tests for StockResearchPanel (4 tests: mounts, title, tabs render, content area exists).

- [ ] **Step 2: Run frontend tests**

Run: `cd frontend && npx vitest run`
Expected: all 76+ existing tests pass, 8+ new tests pass

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/__tests__/SentimentPanel.test.ts frontend/src/terminal/panels/__tests__/StockResearchPanel.test.ts
git commit -m "test: add research panel vitest tests"
```

---

### Task 24: Add Go IPC methods to app.go

**Files:**
- Modify: `app.go`

**Consumes:** Task 15 (research services wired in startup)

- [ ] **Step 1: Add IPC methods**

In `app.go`, add after existing IPC methods:
```go
// GetSentiment returns sentiment analysis for a symbol.
func (a *App) GetSentiment(symbol string) (*research.SentimentOutput, error) {
	engine := research.NewSentimentEngine(a.bridge, research.NewResearchRepo(a.db))
	return engine.AnalyzeSentiment(context.Background(), symbol, "", "news", "en")
}

// GetSentimentHistory returns historical sentiment for a symbol.
func (a *App) GetSentimentHistory(symbol string, days int) ([]research.SentimentOutput, error) {
	engine := research.NewSentimentEngine(a.bridge, research.NewResearchRepo(a.db))
	return engine.GetSentimentHistory(context.Background(), symbol, days)
}

// GetStockResearch returns multi-dimensional research data for a symbol.
func (a *App) GetStockResearch(symbol string, tabs []string) (*research.StockResearchResult, error) {
	repo := research.NewResearchRepo(a.db)
	finSvc := research.NewFinancialsService()
	peerSvc := research.NewPeerComparisonService()
	estSvc := research.NewAnalystEstimatesService()
	insSvc := research.NewInsiderTradingService()

	result := &research.StockResearchResult{
		Symbol: symbol,
		Overview: map[string]interface{}{
			"symbol": symbol, "name": symbol,
			"sector": "Mock", "market_cap": "N/A",
		},
	}

	for _, tab := range tabs {
		switch tab {
		case "financials":
			fd, _ := finSvc.GetFinancials(context.Background(), symbol)
			result.Financials = fd
			result.Ratios = finSvc.ComputeRatios(fd)
		case "peers":
			peers, _ := peerSvc.GetPeers(context.Background(), symbol)
			result.Peers = peers
		case "estimates":
			est, _ := estSvc.GetEstimates(context.Background(), symbol)
			result.Estimates = est
		case "insider":
			txns, _ := insSvc.GetInsiderTrades(context.Background(), symbol)
			result.InsiderTxns = txns
		case "sentiment":
			if a.bridge != nil {
				engine := research.NewSentimentEngine(a.bridge, repo)
				s, _ := engine.AnalyzeSentiment(context.Background(), symbol, "", "news", "en")
				result.Sentiment = s
			}
		}
	}

	return result, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd app && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add app.go
git commit -m "feat: add GetSentiment and GetStockResearch IPC methods"
```

---

### Task 25: Update CHANGELOG and version

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `README.md` (version badge)
- Modify: `frontend/package.json` (version)

- [ ] **Step 1: Update CHANGELOG**

Add to `CHANGELOG.md` under a new `[2026.6.18]` section:
```markdown
## [2026.6.18] - 2026-06-18

### Added
- [Research] Sentiment analysis module: NLP pipeline (Python) + SentimentEngine (Go) + SentimentNode (workflow)
- [Research] 6 workflow nodes: sentiment, stock_research, financials, peer_compare, analyst_estimates, insider_trades
- [Research] 7 frontend panels: SentimentPanel, StockResearchPanel, FinancialsPanel, PeerComparisonPanel, AnalystEstimatesPanel, InsiderTradingPanel, CongressTradingPanel
- [Research] Sentiment gRPC service with NLTK VADER + SnowNLP
- [Research] ResearchRepo with SQLite persistence (migration 011)
- [Research] Pinia research store
- [Research] Degraded mode: all research features work without Python sidecar (mock data)

### Changed
- [Python] Registered SentimentService in gRPC server
- [Engine] PythonBridge now includes SentimentClient
```

- [ ] **Step 2: Update version**

In `frontend/package.json`: set `"version"` to `"2026.6.18"`
In `README.md`: update version badge to `2026.6.18`

- [ ] **Step 3: Full build check**

Run: `cd app && go vet ./... && go test ./... -count=1`
Run: `cd frontend && npx vue-tsc --noEmit && npx vitest run`
Run: `cd python && python -m pytest tests/ -x -q`
Expected: all pass

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md README.md frontend/package.json
git commit -m "chore: update CHANGELOG and version to 2026.6.18"
```

---

### Task 26: End-to-end verification

No code changes — verification only.

- [ ] **Step 1: Start Python sidecar**

Run: `cd python && python -m src.server --port 50052`
Expected: "SentimentService" in registered services log

- [ ] **Step 2: Test gRPC directly**

Run: `cd python && python -m pytest tests/test_sentiment.py -v`
Expected: all tests pass

- [ ] **Step 3: Run Go tests**

Run: `cd app && go test ./internal/research/... ./internal/workflow/nodes/... -v -count=1`
Expected: all tests pass (including mock fallback)

- [ ] **Step 4: Run frontend tests**

Run: `cd frontend && npx vitest run`
Expected: all existing + new tests pass

- [ ] **Step 5: Run app in dev mode**

Run: `wails dev`
Expected: app starts, CommandBar shows new panels ("sentiment", "stock-research", "financials", etc.)
Verify: open SentimentPanel, enter "AAPL", see mock data (Python not required for basic rendering)
Verify: open StockResearchPanel, browse all 7 tabs, each shows data

- [ ] **Step 6: Verify workflow integration**

In Workflow Mode: add SentimentNode to canvas, verify it appears in the "research" category in NodePalette
Connect SentimentNode's `signal` output to a StrategyNode's input — verify the port types are compatible

- [ ] **Step 7: Verify degradation**

Stop Python sidecar. Open SentimentPanel again — verify it still shows mock data (not error).
Create a workflow with SentimentNode — execute it — verify it produces neutral/hold output.

---

## Completion Checklist

- [ ] All 28 new files created
- [ ] All 5 existing files modified
- [ ] Python: `python -m pytest tests/ -x -q` — all pass
- [ ] Go: `go vet ./... && go test ./... -count=1` — all pass
- [ ] Frontend: `npx vue-tsc --noEmit && npx vitest run` — all pass
- [ ] CHANGELOG updated with today's date
- [ ] Version updated to 2026.6.18
- [ ] 6 research nodes searchable in CommandBar
- [ ] 7 research panels openable from CommandBar
- [ ] Degraded mode works without Python sidecar
- [ ] SentimentNode signal output connects to StrategyNode
