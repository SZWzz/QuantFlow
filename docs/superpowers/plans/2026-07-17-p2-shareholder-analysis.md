# ShareholderAnalysis Implementation Plan

**Goal:** Top-10 circulating shareholders evolution, institutional holdings changes, northbound ownership trend.

**Architecture:** New `EastMoneyShareholderAdapter` fetches shareholder data from EastMoney API. `ShareholderPanel.vue` renders 3 tables.

**New data source needed:** EastMoney shareholder API `https://datacenter.eastmoney.com/api/data/v1/get` → `RPT_F10_SHAREHOLDER_TOP10` (top 10 shareholders) / `RPT_F10_SHAREHOLDER_INSTITUTION` (institutional holdings).

---

### Task 1: EastMoneyShareholderAdapter + Go test

**Files:**
- Create: `internal/market/adapters/eastmoney_shareholder.go`
- Test: `internal/market/adapters/eastmoney_shareholder_test.go`

```go
// internal/market/adapters/eastmoney_shareholder.go
type ShareholderRecord struct {
    Name         string  `json:"name"`
    Type         string  `json:"type"`         // 基金/保险/QFII/个人...
    Shares       float64 `json:"shares"`       // 持股数
    Pct          float64 `json:"pct"`          // 占总股本%
    Change       float64 `json:"change"`       // 较上期变动(股)
    MarketValue  float64 `json:"market_value"` // 持股市值
    ReportDate   string  `json:"report_date"`
}

type EastMoneyShareholderAdapter struct { client *http.Client }

func (a *EastMoneyShareholderAdapter) FetchTop10Holders(ctx context.Context, symbol string) ([]ShareholderRecord, error) {
    // EastMoney datacenter API: RPT_F10_SHAREHOLDER_TOP10
}

func (a *EastMoneyShareholderAdapter) FetchInstitutionalTrend(ctx context.Context, symbol string) ([]ShareholderRecord, error) {
    // Aggregate institutional holdings across quarters
}
```

**Commit:** `feat(adapters): add EastMoney shareholder data adapter`

---

### Task 2: ShareholderPanel.vue

**Files:**
- Create: `frontend/src/terminal/panels/ShareholderPanel.vue`
- Modify: registry.ts → register `shareholder-analysis`

**UI:** Three tab tables: (1) 十大流通股东（当前报告期，按持股比例排序），(2) 机构持仓变动（季度趋势表），(3) 北向持股趋势（日频，复用 THSNorthboundAdapter 数据）。点击机构名称高亮该机构所有持仓。

**Commit:** `feat(frontend): add ShareholderAnalysis panel`

---

### Task 3: IPC + wire

**Files:**
- Create: `app_shareholder.go`
- Modify: `frontend/src/lib/wails.ts`
- Modify: `app_startup.go` / `app.go` / `adapters`

**Commit:** `feat(backend+frontend): wire ShareholderAnalysis IPC`
