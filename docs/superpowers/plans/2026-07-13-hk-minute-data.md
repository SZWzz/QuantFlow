# 港股分时数据双通道 — 实施计划

> **前置:** `docs/specs/2026-07-13-hk-minute-data.md`

**目标:** 港股分时数据现有 Yahoo 单通道在中国 geo-block 后完全不可用。新增 AKShare（Python sidecar，免费）和 QOS（付费，需 API key）两个通道，Yahoo 作为最后回退。

**数据流:**

```
GetMinuteLine("00700") → FetchMinuteWithFallback("HK", "00700")
  → akshare_hk_minute.FetchMinuteLine (Python gRPC, 免费)
  → qos.FetchMinuteLine (HTTP, 需 key)
  → yahoo.FetchMinuteLine (geo-block 时失败)
```

**技术栈:** Go 1.25+ (Wails v3), Vue 3 + TypeScript, Python 3.12+ (gRPC), SQLite WAL

---

## Phase A: AKShare Python + Go 适配器（主通道）

### Task A1: Python 侧 `_fetch_akshare_hk_minute`

**文件:** `python/src/data/fetcher.py`
**动作:** 在文件末尾（`# ---------------------------------------------------------------------------` 前）添加函数

```python
_FETCH_AKSHARE_HK_MINUTE_CACHE: dict[str, list[dict]] = {}
_FETCH_AKSHARE_HK_MINUTE_CACHE_TS: dict[str, float] = {}

def _fetch_akshare_hk_minute(symbol: str) -> list[dict]:
    """Fetch HK stock minute data via akshare.stock_hk_hist_min_em.
    
    Returns [{time, price, volume}, ...] for today, ordered by time ascending.
    Cached for 60s per symbol to avoid redundant API calls.
    """
    now = time.time()
    cached_ts = _FETCH_AKSHARE_HK_MINUTE_CACHE_TS.get(symbol, 0)
    if now - cached_ts < 60 and symbol in _FETCH_AKSHARE_HK_MINUTE_CACHE:
        return _FETCH_AKSHARE_HK_MINUTE_CACHE[symbol]

    today = datetime.now().strftime("%Y%m%d")
    try:
        ak = importlib.import_module("akshare")
    except ImportError:
        logger.warning("akshare not installed, cannot fetch HK minute data")
        return []

    try:
        df = ak.stock_hk_hist_min_em(
            symbol=symbol,
            period="1",
            start_date=today,
            end_date=today,
        )
    except Exception as exc:
        logger.warning("akshare HK minute failed for %s: %s", symbol, exc)
        return []

    if df is None or (hasattr(df, "empty") and df.empty):
        return []

    col_map = {}
    for c in df.columns:
        cl = str(c).strip().lower()
        if "时间" in cl:
            col_map[c] = "time"
        elif "开盘" in cl:
            col_map[c] = "open"
        elif "收盘" in cl:
            col_map[c] = "close"
        elif "最高" in cl:
            col_map[c] = "high"
        elif "最低" in cl:
            col_map[c] = "low"
        elif "成交量" in cl:
            col_map[c] = "volume"
        elif "成交额" in cl:
            col_map[c] = "amount"
    df = df.rename(columns=col_map)

    if "close" not in df.columns:
        return []

    bars = []
    for _, row in df.iterrows():
        bar = {"symbol": symbol}
        dt = str(row.get("time", ""))
        if dt:
            bar["date"] = str(dt)[:19]
        for k in ("open", "high", "low", "close", "volume", "amount"):
            try:
                bar[k] = float(row.get(k, 0)) if row.get(k) is not None else 0.0
            except (ValueError, TypeError):
                bar[k] = 0.0
        bars.append(bar)

    bars.sort(key=lambda b: b.get("date", ""))
    _FETCH_AKSHARE_HK_MINUTE_CACHE[symbol] = bars
    _FETCH_AKSHARE_HK_MINUTE_CACHE_TS[symbol] = now
    return bars
```

- [ ] **Step A1.1:** 添加函数到 `fetcher.py`
- [ ] **Step A1.2:** 验证 Python 语法: `python -c "import ast; ast.parse(open('python/src/data/fetcher.py').read())"`

---

### Task A2: Python gRPC 处理 `hk_minute` 数据类

**文件:** `python/src/data/fetcher.py`
**动作:** 在 `_handle_akshare` 中 route 查找前插入 `hk_minute` 特殊分支

在 `fetch.py` 的 `_handle_akshare` 方法中找到：
```python
route = self._AKSHARE_ROUTES.get(data_type)
if route is None:
    return data_pb2.FetchDataResponse(...)
```

替换为：
```python
if data_type == "hk_minute":
    symbol = symbols[0] if symbols else ""
    if not symbol:
        return data_pb2.FetchDataResponse(error="hk_minute requires exactly one symbol")
    loop = asyncio.get_event_loop()
    bars = await loop.run_in_executor(None, lambda: _fetch_akshare_hk_minute(symbol))
    return data_pb2.FetchDataResponse(
        data=json.dumps(bars, ensure_ascii=False).encode("utf-8")
    )

route = self._AKSHARE_ROUTES.get(data_type)
if route is None:
    return data_pb2.FetchDataResponse(
        error=f"AKShare: unknown data_type '{data_type}'..."
    )
```

- [ ] **Step A2.1:** 修改 `_handle_akshare`，在 route 查找前插入 hk_minute 分支
- [ ] **Step A2.2:** 验证 Python 语法 `python -c "import ast; ast.parse(open('python/src/data/fetcher.py').read())"`

---

### Task A3: Python 测试

**文件:** `python/tests/test_hk_minute.py`
**动作:** 新建测试文件

```python
"""Tests for HK minute data fetching via AKShare."""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parent.parent / "src"
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

import json
from unittest.mock import patch, MagicMock
import pandas as pd
import pytest


def test_fetch_akshare_hk_minute_empty_df():
    """Return empty list when DataFrame is empty."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    mock_df = pd.DataFrame()
    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        mock_ak = MagicMock()
        mock_ak.stock_hk_hist_min_em.return_value = mock_df
        mock_import.return_value = mock_ak

        result = _fetch_akshare_hk_minute("00700")
        assert result == []


def test_fetch_akshare_hk_minute_parses_columns():
    """Parse Chinese column names to standard format."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    df = pd.DataFrame({
        "时间": ["2026-07-13 09:30:00", "2026-07-13 09:31:00"],
        "开盘": [100.0, 101.0],
        "收盘": [100.5, 101.5],
        "最高": [101.0, 102.0],
        "最低": [99.5, 100.5],
        "成交量": [10000, 15000],
        "成交额": [1005000, 1522500],
    })

    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        mock_ak = MagicMock()
        mock_ak.stock_hk_hist_min_em.return_value = df
        mock_import.return_value = mock_ak

        result = _fetch_akshare_hk_minute("00700")
        assert len(result) == 2
        assert result[0]["symbol"] == "00700"
        assert result[0]["close"] == 100.5
        assert result[0]["volume"] == 10000
        assert result[0]["amount"] == 1005000
        assert result[0]["date"] == "2026-07-13 09:30:00"


def test_fetch_akshare_hk_minute_cache():
    """Return cached result within 60s."""
    from src.data.fetcher import _fetch_akshare_hk_minute, _FETCH_AKSHARE_HK_MINUTE_CACHE

    # Prime cache
    _FETCH_AKSHARE_HK_MINUTE_CACHE["00700"] = [{"time": "09:30", "price": 100.0}]
    from src.data.fetcher import _FETCH_AKSHARE_HK_MINUTE_CACHE_TS
    import time
    _FETCH_AKSHARE_HK_MINUTE_CACHE_TS["00700"] = time.time()

    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        result = _fetch_akshare_hk_minute("00700")
        mock_import.assert_not_called()
        assert len(result) == 1


def test_fetch_akshare_hk_minute_import_error():
    """Return empty list when akshare is not installed."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    with patch("src.data.fetcher.importlib.import_module", side_effect=ImportError):
        result = _fetch_akshare_hk_minute("00700")
        assert result == []
```

- [ ] **Step A3.1:** 新建 `python/tests/test_hk_minute.py`
- [ ] **Step A3.2:** 运行测试 `cd python && python -m pytest tests/test_hk_minute.py -x -q 2>&1 | tail -10`
- [ ] **Step A3.3:** 提交 `git add python/src/data/fetcher.py python/tests/test_hk_minute.py CHANGELOG.md && git commit -m "feat(python): add HK minute data via akshare.stock_hk_hist_min_em"`

---

### Task A4: Go `AKShareMinuteAdapter` 适配器

**文件:** `internal/market/adapters/akshare_minuteline.go`（新建）

```go
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"quantflow/internal/market"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// akshareMinuteCooldown prevents hammering the free Python sidecar.
const akshareMinuteCooldown = 3 * time.Second

// AKShareMinuteAdapter fetches HK minute data via Python akshare gRPC sidecar.
// Implements market.MinuteLineProvider.
type AKShareMinuteAdapter struct {
	dataClient *python.DataClient
	lastFetch  map[string]time.Time
	mu         sync.Mutex
}

// NewAKShareMinuteAdapter creates a new adapter.
// dataClient may be nil (degrade gracefully).
func NewAKShareMinuteAdapter(dataClient *python.DataClient) *AKShareMinuteAdapter {
	return &AKShareMinuteAdapter{
		dataClient: dataClient,
		lastFetch:  make(map[string]time.Time),
	}
}

func (a *AKShareMinuteAdapter) Name() string      { return "akshare_hk_minute" }
func (a *AKShareMinuteAdapter) Markets() []string  { return []string{"HK"} }
func (a *AKShareMinuteAdapter) RequiresAuth() bool { return false }

func (a *AKShareMinuteAdapter) IsAvailable(ctx context.Context) bool {
	if a.dataClient == nil {
		return false
	}
	return true
}

func (a *AKShareMinuteAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	if a.dataClient == nil {
		return nil, fmt.Errorf("akshare_hk_minute: Python sidecar not connected")
	}

	a.mu.Lock()
	if last, ok := a.lastFetch[symbol]; ok && time.Since(last) < akshareMinuteCooldown {
		a.mu.Unlock()
		return nil, nil // cooldown active
	}
	a.lastFetch[symbol] = time.Now()
	a.mu.Unlock()

	resp, err := a.dataClient.FetchData(context.Background(), &pb.FetchDataRequest{
		Source:   "akshare",
		DataType: "hk_minute",
		Symbols:  []string{symbol},
	})
	if err != nil {
		return nil, fmt.Errorf("akshare_hk_minute: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("akshare_hk_minute: %s", resp.Error)
	}

	var raw []struct {
		Time   string  `json:"time"`
		Date   string  `json:"date,omitempty"`
		Price  float64 `json:"close"`
		Volume float64 `json:"volume"`
		Amount float64 `json:"amount,omitempty"`
	}
	if err := json.Unmarshal(resp.Data, &raw); err != nil {
		return nil, fmt.Errorf("akshare_hk_minute parse: %w", err)
	}

	ticks := make([]market.MinuteTick, 0, len(raw))
	for _, r := range raw {
		// Extract time from date+time string like "2026-07-13 09:30:00"
		t := r.Time
		if len(r.Date) >= 16 {
			t = r.Date[11:16]
		} else if len(r.Time) >= 16 {
			t = r.Time[11:16]
		}
		ticks = append(ticks, market.MinuteTick{
			Time:   t,
			Price:  r.Price,
			Volume: r.Volume,
			Amount: r.Amount,
		})
	}
	return ticks, nil
}
```

- [ ] **Step A4.1:** 创建 `akshare_minuteline.go`
- [ ] **Step A4.2:** 编译验证 `go build ./internal/market/adapters/`

---

### Task A5: 注册到 HK 分钟链

**文件:** `internal/market/registry.go` + `app_market.go`

```go
// internal/market/registry.go:48 — MinuteChains["HK"]
"HK": {
    "stock": {"akshare_hk_minute", "qos", "yahoo"},
},
```

```go
// app_market.go — registerMarketAdapters()
a.marketReg.Register(adapters.NewAKShareMinuteAdapter(dataClient))
```

- [ ] **Step A5.1:** 修改 `registry.go` MinuteChains
- [ ] **Step A5.2:** 修改 `app_market.go` registerMarketAdapters
- [ ] **Step A5.3:** 编译 `go build ./...`

---

### Task A6: Go 适配器测试

**文件:** `internal/market/adapters/akshare_minuteline_test.go`（新建）

```go
package adapters

import (
	"context"
	"encoding/json"
	"testing"

	"quantflow/internal/market"
	pb "quantflow/internal/python/proto"
)

// mockDataClient implements a minimal FetchData for testing.
type mockDataClient struct {
	fn func(ctx context.Context, req *pb.FetchDataRequest) (*pb.FetchDataResponse, error)
}

func (m *mockDataClient) FetchData(ctx context.Context, req *pb.FetchDataRequest) (*pb.FetchDataResponse, error) {
	return m.fn(ctx, req)
}

func TestAKShareMinuteAdapter_Name(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	if a.Name() != "akshare_hk_minute" {
		t.Fatalf("expected akshare_hk_minute, got %s", a.Name())
	}
}

func TestAKShareMinuteAdapter_Markets(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	mkts := a.Markets()
	if len(mkts) != 1 || mkts[0] != "HK" {
		t.Fatalf("expected [HK], got %v", mkts)
	}
}

func TestAKShareMinuteAdapter_IsAvailable_NilClient(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	if a.IsAvailable(context.Background()) {
		t.Fatal("expected false when client is nil")
	}
}

func TestAKShareMinuteAdapter_IsAvailable_WithClient(t *testing.T) {
	a := NewAKShareMinuteAdapter(&mockDataClient{})
	if !a.IsAvailable(context.Background()) {
		t.Fatal("expected true when client is set")
	}
}

func TestAKShareMinuteAdapter_FetchMinuteLine_Success(t *testing.T) {
	raw := []map[string]any{
		{"date": "2026-07-13 09:30:00", "close": 100.5, "volume": 10000.0, "amount": 1005000.0},
		{"date": "2026-07-13 09:31:00", "close": 101.0, "volume": 15000.0, "amount": 1515000.0},
	}
	data, _ := json.Marshal(raw)

	client := &mockDataClient{
		fn: func(ctx context.Context, req *pb.FetchDataRequest) (*pb.FetchDataResponse, error) {
			if req.Source != "akshare" || req.DataType != "hk_minute" {
				t.Fatalf("unexpected request: %s/%s", req.Source, req.DataType)
			}
			return &pb.FetchDataResponse{Data: data}, nil
		},
	}

	a := NewAKShareMinuteAdapter(client)
	ticks, err := a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ticks) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(ticks))
	}
	if ticks[0].Time != "09:30" || ticks[0].Price != 100.5 || ticks[0].Volume != 10000 {
		t.Fatalf("unexpected tick 0: %+v", ticks[0])
	}
}

func TestAKShareMinuteAdapter_Cooldown(t *testing.T) {
	a := NewAKShareMinuteAdapter(&mockDataClient{
		fn: func(ctx context.Context, req *pb.FetchDataRequest) (*pb.FetchDataResponse, error) {
			return &pb.FetchDataResponse{
				Data: []byte("[]"),
			}, nil
		},
	})

	// First call: should proceed
	ticks, err := a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = ticks

	// Second call (immediate): cooldown should return nil, nil
	ticks, err = a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error on cooldown: %v", err)
	}
	if ticks != nil {
		t.Fatal("expected nil ticks during cooldown")
	}
}
```

- [ ] **Step A6.1:** 新建测试文件
- [ ] **Step A6.2:** 运行测试 `go test ./internal/market/adapters/ -v -count=1 -run TestAKShare 2>&1 | tail -20`
- [ ] **Step A6.3:** 提交 `git add internal/market/adapters/akshare_minuteline.go internal/market/adapters/akshare_minuteline_test.go internal/market/registry.go app_market.go CHANGELOG.md && git commit -m "feat(hk-minute): add AKShare Python sidecar minute adapter — free HK minute data via EastMoney"`

---

## Phase B: QOS API 适配器（备用付费通道）

### Task B1: QOS 适配器

**文件:** `internal/market/adapters/qos_minuteline.go`（新建）

```go
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"quantflow/internal/market"
)

// qosMinuteCooldown prevents hitting QOS free-tier rate limits.
const qosMinuteCooldown = 3 * time.Second

// QOSMinuteAdapter fetches HK minute data via QOS API.
// Implements market.MinuteLineProvider.
type QOSMinuteAdapter struct {
	client    *http.Client
	apiKey    string
	baseURL   string
	lastFetch map[string]time.Time
	mu        sync.Mutex
}

// QOSConfig holds configuration for the QOS adapter.
type QOSConfig struct {
	APIKey  string
	BaseURL string
}

// NewQOSMinuteAdapter creates a new QOS adapter.
func NewQOSMinuteAdapter(cfg QOSConfig) *QOSMinuteAdapter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.qos.hk"
	}
	return &QOSMinuteAdapter{
		client:    &http.Client{Timeout: 10 * time.Second},
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		lastFetch: make(map[string]time.Time),
	}
}

func (a *QOSMinuteAdapter) Name() string                { return "qos" }
func (a *QOSMinuteAdapter) Markets() []string            { return []string{"HK"} }
func (a *QOSMinuteAdapter) RequiresAuth() bool           { return true }
func (a *QOSMinuteAdapter) SetAPIKey(key string)         { a.apiKey = key }

func (a *QOSMinuteAdapter) IsAvailable(ctx context.Context) bool {
	if a.apiKey == "" {
		return false
	}
	return true
}

// qosKlineReq mirrors the QOS API request body.
type qosKlineReq struct {
	Key      string          `json:"key"`
	KlineReqs []qosSingleReq `json:"kline_reqs"`
}

type qosSingleReq struct {
	Code string `json:"c"`   // "HK:700"
	Count int   `json:"co"`  // 240 = number of 1-min bars
	Adjust int  `json:"a"`   // 0 = no adjust
	Ktype int   `json:"kt"`  // 1 = 1min
}

// qosKlineRespItem is a single tick in the QOS response "k" array.
type qosKlineRespItem struct {
	T string  `json:"t"` // time "0910" (HHMM)
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
	V float64 `json:"v"` // volume
	// ... other fields ignored
}

func (a *QOSMinuteAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("qos: API key not configured")
	}

	a.mu.Lock()
	if last, ok := a.lastFetch[symbol]; ok && time.Since(last) < qosMinuteCooldown {
		a.mu.Unlock()
		return nil, nil
	}
	a.lastFetch[symbol] = time.Now()
	a.mu.Unlock()

	// Convert symbol to QOS format: "00700" → "HK:700"
	qosSymbol := toQOSSymbol(symbol)

	reqBody := qosKlineReq{
		Key: a.apiKey,
		KlineReqs: []qosSingleReq{
			{Code: qosSymbol, Count: 240, Adjust: 0, Ktype: 1},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	url := a.baseURL + "/kline"

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("qos: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("qos: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("qos: HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("qos: read error: %w", err)
	}

	var qosResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			K []qosKlineRespItem `json:"k"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &qosResp); err != nil {
		return nil, fmt.Errorf("qos: parse error: %w", err)
	}
	if qosResp.Code != 0 {
		return nil, fmt.Errorf("qos: API error %d: %s", qosResp.Code, qosResp.Msg)
	}
	if len(qosResp.Data) == 0 {
		return nil, fmt.Errorf("qos: no data for %s", symbol)
	}

	ticks := make([]market.MinuteTick, 0, len(qosResp.Data[0].K))
	for _, k := range qosResp.Data[0].K {
		// QOS time format "0910" → "09:10"
		t := k.T
		if len(t) == 4 {
			t = t[:2] + ":" + t[2:]
		}
		ticks = append(ticks, market.MinuteTick{
			Time:   t,
			Price:  k.C,
			Volume: k.V,
		})
	}
	return ticks, nil
}

// toQOSSymbol converts a HK symbol to QOS format.
// "00700" → "HK:700", "00005" → "HK:5"
func toQOSSymbol(symbol string) string {
	// Strip leading zeros
	i := 0
	for i < len(symbol) && symbol[i] == '0' {
		i++
	}
	num := symbol[i:]
	if num == "" {
		num = "0"
	}
	return "HK:" + num
}
```

- [ ] **Step B1.1:** 创建 `qos_minuteline.go`
- [ ] **Step B1.2:** 编译 `go build ./internal/market/adapters/`

---

### Task B2: QOS 配置 + GetAPIKey

**文件:** `internal/config/config.go` + `app_market.go`

`GetAPIKey("qos")` 已支持 CM-first 模式（CredentialManager 优先、env 兜底），只需在前端设置面板增加输入框。

`app_market.go` 注册时设置 API key：
```go
qosAdpt := adapters.NewQOSMinuteAdapter(adapters.QOSConfig{
    APIKey: a.cfg.GetAPIKey("qos"),
})
a.marketReg.Register(qosAdpt)
```

- [ ] **Step B2.1:** 在 `registerMarketAdapters()` 添加 QOS 注册（app_market.go）
- [ ] **Step B2.2:** 编译 `go build ./...`

---

### Task B3: QOS 前端配置

**文件:** `frontend/src/terminal/panels/BrokerConfig.vue` 或 `frontend/src/terminal/panels/SettingsPanel.vue`

在 SettingsPanel 或 BrokerConfig 增加 QOS API Key 输入框：
```html
<div class="config-section">
  <h4 class="section-title">QOS API Key（港股分时备用）</h4>
  <div class="form-group">
    <label>API Key</label>
    <input v-model="qosApiKey" type="password" class="form-input" placeholder="输入 QOS API Key" />
  </div>
</div>
```

```typescript
const qosApiKey = ref('')

async function loadQOSKey() {
  try {
    const app = (window as any).go?.main?.App
    const cred = await app?.GetCredential('qos_api_key')
    if (cred?.keys?.api_key) qosApiKey.value = cred.keys.api_key
  } catch { /* ignore */ }
}

async function saveQOSKey() {
  const app = (window as any).go?.main?.App
  if (app?.SaveCredential) {
    await app.SaveCredential('qos_api_key', 'api', { api_key: qosApiKey.value })
  }
}
```

- [ ] **Step B3.1:** 选择合适面板添加 QOS 配置（SettingsPanel 或 BrokerConfig）
- [ ] **Step B3.2:** 编译 `cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5`

---

### Task B4: QOS 适配器测试

**文件:** `internal/market/adapters/qos_minuteline_test.go`（新建）

```go
package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQOSMinuteAdapter_Name(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	if a.Name() != "qos" {
		t.Fatalf("expected qos, got %s", a.Name())
	}
}

func TestQOSMinuteAdapter_RequiresAuth(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	if !a.RequiresAuth() {
		t.Fatal("expected RequiresAuth=true")
	}
}

func TestQOSMinuteAdapter_IsAvailable_NoKey(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	if a.IsAvailable(context.Background()) {
		t.Fatal("expected false with no API key")
	}
}

func TestQOSMinuteAdapter_IsAvailable_WithKey(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{APIKey: "test-key"})
	if !a.IsAvailable(context.Background()) {
		t.Fatal("expected true with API key")
	}
}

func TestQOSMinuteAdapter_FetchMinuteLine_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if req["key"] != "test-key" {
			t.Fatal("wrong API key")
		}
		resp := map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": []map[string]any{
				{"k": []map[string]any{
					{"t": "0930", "o": 100.0, "h": 101.0, "l": 99.5, "c": 100.5, "v": 10000},
					{"t": "0931", "o": 101.0, "h": 102.0, "l": 100.5, "c": 101.5, "v": 15000},
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	a := NewQOSMinuteAdapter(QOSConfig{APIKey: "test-key", BaseURL: server.URL})
	ticks, err := a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ticks) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(ticks))
	}
	if ticks[0].Time != "09:30" || ticks[0].Price != 100.5 || ticks[0].Volume != 10000 {
		t.Fatalf("unexpected tick 0: %+v", ticks[0])
	}
}

func TestToQOSSymbol(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"00700", "HK:700"},
		{"00005", "HK:5"},
		{"700", "HK:700"},
		{"00001", "HK:1"},
		{"09988", "HK:9988"},
	}
	for _, tt := range tests {
		got := toQOSSymbol(tt.in)
		if got != tt.out {
			t.Errorf("toQOSSymbol(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
```

- [ ] **Step B4.1:** 新建测试文件
- [ ] **Step B4.2:** 运行测试 `go test ./internal/market/adapters/ -v -count=1 -run TestQOS 2>&1 | tail -20`
- [ ] **Step B4.3:** 提交 `git add internal/market/adapters/qos_minuteline.go internal/market/adapters/qos_minuteline_test.go app_market.go frontend/src/terminal/panels/*.vue CHANGELOG.md && git commit -m "feat(hk-minute): add QOS API minute adapter — paid backup for HK minute data"`

---

## 验证清单

- [ ] `go test ./...` 全部通过
- [ ] `cd frontend && npx vue-tsc --noEmit` 无新错误
- [ ] `cd frontend && npx vitest run` 通过
- [ ] `cd python && python -m pytest tests/test_hk_minute.py -x -q` 通过
- [ ] AKShare HK 分时返回正确格式数据
- [ ] QOS API Key 可保存/加载
- [ ] 港股分时在前端 CandlestickPanel 正常显示
