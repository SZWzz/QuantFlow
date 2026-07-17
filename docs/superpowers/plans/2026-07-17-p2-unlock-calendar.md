# UnlockCalendar Implementation Plan

**Goal:** Restricted share unlock calendar + impact estimation. Warns when unlock ratio > 5% of float.

**Architecture:** New `EastMoneyUnlockAdapter` fetches unlock data. `UnlockCalendarPanel.vue` renders timeline.

**New data source needed:** EastMoney unlock API `RPT_F10_SHAREHOLDER_UNLOCK`.

---

### Task 1: EastMoneyUnlockAdapter

**Files:**
- Create: `internal/market/adapters/eastmoney_unlock.go`

```go
type UnlockEvent struct {
    Symbol      string  `json:"symbol"`
    Name        string  `json:"name"`
    UnlockDate  string  `json:"unlock_date"`
    UnlockShares float64 `json:"unlock_shares"` // 解禁数量(万股)
    UnlockPct   float64 `json:"unlock_pct"`     // 占总股本%
    FloatRatio  float64 `json:"float_ratio"`    // 占流通股%
    MarketValue float64 `json:"market_value"`   // 解禁市值(亿)
    LockPeriod  int     `json:"lock_period"`    // 限售月数
}

type EastMoneyUnlockAdapter struct { client *http.Client }

func (a *EastMoneyUnlockAdapter) FetchUpcoming(ctx context.Context, days int) ([]UnlockEvent, error) {
    // EastMoney: RPT_F10_SHAREHOLDER_UNLOCK, filter by unlock_date in [today, today+days]
}
```

**Commit:** `feat(adapters): add EastMoney unlock calendar adapter`

---

### Task 2: UnlockCalendarPanel.vue

**Files:**
- Create: `frontend/src/terminal/panels/UnlockCalendarPanel.vue`
- Modify: registry.ts → register `unlock-calendar`

**UI:** Timeline/list grouped by week. Each entry: symbol, name, unlock shares, unlock pct, float ratio, estimated impact (¥). Warning badge when UnlockPct > 5%. Month selector for forward-looking view.

**Commit:** `feat(frontend): add UnlockCalendar panel with impact warnings`

---

### Task 3: IPC + wire

**Commit:** `feat(backend+frontend): wire UnlockCalendar IPC`
