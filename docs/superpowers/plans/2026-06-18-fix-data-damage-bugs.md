# Implementation Plan: Fix 4 Data-Damage Bugs

> Spec: [docs/specs/2026-06-18-fix-data-damage-bugs.md](../specs/2026-06-18-fix-data-damage-bugs.md)
> Date: 2026-06-18

---

## Task 1: Fix #4 — TuShare API 解析错位

### Step 1.1: Add zipFieldsAndItems + fix callAPI

**File:** `internal/market/adapters/tushare.go`

In `callAPI`, after `result.Code != 0` check and before `return &result, nil`, add:

```go
// TuShare returns data as fields+items (parallel arrays), not maps.
// Convert to the map form expected by FetchQuote/FetchOHLCV.
if len(result.Items) == 0 && len(result.Data.Fields) > 0 {
    result.Items = zipFieldsAndItems(result.Data.Fields, result.Data.Items)
}
```

Add the helper function at the bottom of the file:

```go
// zipFieldsAndItems converts TuShare's fields+items parallel-array format
// into []map[string]any, matching the API shape expected by callers.
func zipFieldsAndItems(fields []string, items [][]any) []map[string]any {
    result := make([]map[string]any, 0, len(items))
    for _, row := range items {
        if len(row) < len(fields) {
            continue
        }
        m := make(map[string]any, len(fields))
        for i, f := range fields {
            m[f] = row[i]
        }
        result = append(result, m)
    }
    return result
}
```

### Step 1.2: Add unit test

**File:** `internal/market/adapters/tushare_test.go` — add test:

```go
func TestZipFieldsAndItems(t *testing.T) {
    fields := []string{"ts_code", "trade_date", "close"}
    items := [][]any{
        {"000001.SZ", "20240601", 12.5},
        {"600519.SH", "20240601", 1780.0},
    }
    result := zipFieldsAndItems(fields, items)
    if len(result) != 2 {
        t.Fatalf("len = %d, want 2", len(result))
    }
    if result[0]["close"] != 12.5 {
        t.Errorf("result[0][\"close\"] = %v, want 12.5", result[0]["close"])
    }
    if result[1]["ts_code"] != "600519.SH" {
        t.Errorf("result[1][\"ts_code\"] = %v", want "600519.SH")
    }
}

func TestZipFieldsAndItems_Empty(t *testing.T) {
    result := zipFieldsAndItems(nil, nil)
    if len(result) != 0 {
        t.Errorf("len = %d, want 0", len(result))
    }
}

func TestZipFieldsAndItems_MismatchedRow(t *testing.T) {
    fields := []string{"a", "b", "c"}
    items := [][]any{{"x"}} // too short
    result := zipFieldsAndItems(fields, items)
    if len(result) != 0 {
        t.Errorf("len = %d, want 0 (short row skipped)", len(result))
    }
}
```

### Step 1.3: Verify + commit

```bash
go build ./... && go test ./internal/market/adapters/... -v -run TestZip -count=1
```

Commit: `fix(market): parse TuShare data.fields+data.items parallel-array format`

---

## Task 2: Fix #6 — numpy import

### Step 2.1: Add import

**File:** `python/src/ml/engine.py`

Add `import numpy as np` after line 8 (`import pyarrow as pa`).

### Step 2.2: Verify

```bash
python -c "from src.ml.engine import MLService; print('OK')"
```

### Step 2.3: Commit

`fix(python): add missing numpy import in ML engine.py`

---

## Task 3: Fix #7 — NaN→NaN (not 0)

### Step 3.1: Change 0.0 to float('nan')

**File:** `python/src/factor/engine.py`, line 58

Change:
```python
values=[float(v) if not pd.isna(v) else 0.0 for v in values.tolist()],
```
to:
```python
values=[float(v) if not pd.isna(v) else float('nan') for v in values.tolist()],
```

### Step 3.2: Verify

```bash
python -m pytest python/tests/ -x -q 2>&1 | tail -5
```

### Step 3.3: Commit

`fix(python): preserve NaN in factor output instead of converting to 0.0`

---

## Task 4: Fix #8 — Migration transactions

### Step 4.1: Wrap migration in transaction

**File:** `internal/storage/migrate.go`

Change the for loop body (around lines 46-52) from:
```go
if _, err := db.Exec(m.SQL); err != nil {
    return fmt.Errorf("migration %d: %w", m.Version, err)
}

if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (?)", m.Version); err != nil {
    return fmt.Errorf("record migration %d: %w", m.Version, err)
}
```
to:
```go
tx, err := db.Begin()
if err != nil {
    return fmt.Errorf("begin tx for migration %d: %w", m.Version, err)
}

if _, err := tx.Exec(m.SQL); err != nil {
    tx.Rollback()
    return fmt.Errorf("migration %d: %w", m.Version, err)
}

if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", m.Version); err != nil {
    tx.Rollback()
    return fmt.Errorf("record migration %d: %w", m.Version, err)
}

if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit migration %d: %w", m.Version, err)
}
```

### Step 4.2: Verify

```bash
go build ./... && go test ./internal/storage/... -v -count=1
```

### Step 4.3: Commit

`fix(storage): wrap each migration in a transaction — prevent half-applied schema`

---

## Task 5: Update CHANGELOG

Similar to previous batch — add 4 entries under `### Fixed`.

---

**Execution order:** Task 1 → 2 → 3 → 4 → 5 (independent, sequential)
