# EventStudy Implementation Plan

**Goal:** Cumulative abnormal return (CAR) analysis around corporate events — earnings, announcements, policy changes.

**Architecture:** `EventStudyService` computes CAR = stock return − benchmark return over configurable event windows. `EventStudyPanel.vue` renders CAR curve + event calendar picker.

**Data sources (all existing):**
- Stock OHLCV → daily returns
- Index OHLCV → benchmark returns
- Earnings calendar → event dates (`EarningsCalendarPanel` THS data)
- News → custom event dates

---

### Task 1: EventStudyService + CAR computation + Go test

**Files:**
- Create: `internal/research/event_study.go`
- Test: `internal/research/event_study_test.go`

```go
// internal/research/event_study.go
type EventStudyResult struct {
    Symbol    string       `json:"symbol"`
    EventDate string       `json:"event_date"`
    EventType string       `json:"event_type"`
    Window    int          `json:"window"` // days before/after
    CAR       float64      `json:"car"`
    DailyAR   []DailyAR    `json:"daily_ar"`
}

type DailyAR struct {
    Date    string  `json:"date"`
    Day     int     `json:"day"`     // relative to event (negative = before)
    AR      float64 `json:"ar"`      // abnormal return
    CAR     float64 `json:"car"`     // cumulative AR up to this day
    StockR  float64 `json:"stock_r"` // stock daily return
    BenchR  float64 `json:"bench_r"` // benchmark daily return
}

type EventStudyService struct {
    ohlcvCache *market.OHLCVCache
}

func (s *EventStudyService) ComputeEventStudy(
    ctx context.Context,
    symbol, eventDate string,
    window int, // e.g. 10 = [-10, +10]
) (*EventStudyResult, error) {
    // 1. Fetch stock daily returns for [eventDate - window, eventDate + window]
    // 2. Fetch benchmark index daily returns for same range
    // 3. AR = stock_return - bench_return for each day
    // 4. CAR = cumulative sum of AR
}
```

**Test:** Verify CAR computation with known EOD data.

**Commit:** `feat(research): add EventStudyService with CAR computation`

---

### Task 2: EventStudyPanel.vue + ECharts

**Files:**
- Create: `frontend/src/terminal/panels/EventStudyPanel.vue`
- Modify: registry.ts → register `event-study`

**UI:** Symbol + event date inputs. Window slider [-30, +30]. ECharts CAR curve with event day vertical line. Below: CAR stats table (multiple windows: [-1,+1], [-5,+5], [-10,+10]). Event calendar for quick date selection (from earnings calendar data).

**Commit:** `feat(frontend): add EventStudy panel with CAR curve and calendar`

---

### Task 3: IPC + wire

**Files:**
- Create: `app_event_study.go`
- Modify: `frontend/src/lib/wails.ts`
- Modify: `app_startup.go` / `app.go`

**Commit:** `feat(backend+frontend): wire EventStudy IPC`
