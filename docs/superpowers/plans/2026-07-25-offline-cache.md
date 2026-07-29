# Offline Cache Strategy for Panel Data

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add persistent offline cache so key panels (Watchlist, PortfolioSummary, TradeHistory, Market scanners, Settings) show stale data when the Go backend is unreachable, rather than blank/error states. Follows the existing in-memory `CacheEntry` + `fetchWithCache` pattern but persists to SQLite via a new Go binding.

**Architecture:** A new `CacheStore` Go package + SQLite table (`panel_cache`), exposed to frontend via a Wails-bound method `CacheGet`/`CacheSet`. The frontend `fetchWithCache` composable is augmented with an optional persistence layer: on fetch success, write to SQLite; on fetch failure (offline), read from SQLite. The cache entry includes `key`, `data` (JSON blob), `expires_at`, `category` for bulk invalidation.

**Data flow:**
```
Panel → fetchWithCache(key, fetcher)
  ├─ in-memory hit → return (same as today)
  ├─ in-memory miss, Go online → call Go binding → write to SQLite → return
  └─ in-memory miss, Go offline → read SQLite → if !expired return, else stale+flag
```

**Tech Stack:** Go SQLite (via existing storage layer), Wails IPC, TypeScript composable.

---

### Task 1: Create Go cache store package

**Files:**
- Create: `internal/storage/cache_repo.go`
- Test: `internal/storage/cache_repo_test.go`

**Design:**
```go
package storage

import (
	"database/sql"
	"encoding/json"
	"time"
)

type CacheEntry struct {
	Key       string    `json:"key"`
	Data      string    `json:"data"` // JSON blob
	ExpiresAt time.Time `json:"expires_at"`
	Category  string    `json:"category,omitempty"` // for bulk invalidation
}

type CacheRepo struct {
	db *sql.DB
}

func NewCacheRepo(db *sql.DB) *CacheRepo { ... }

func (r *CacheRepo) Get(key string) (*CacheEntry, error) { ... }
func (r *CacheRepo) Set(key string, data string, ttl time.Duration, category string) error { ... }
func (r *CacheRepo) Delete(key string) error { ... }
func (r *CacheRepo) DeleteByCategory(category string) error { ... }
func (r *CacheRepo) CleanExpired() error { ... }
```

- [ ] **Step 1: Add migration for panel_cache table**

Add to `internal/storage/migrate.go`:
```go
{Version: 11, SQL: `
CREATE TABLE IF NOT EXISTS panel_cache (
    key TEXT PRIMARY KEY,
    data TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    category TEXT DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX idx_panel_cache_category ON panel_cache(category);
CREATE INDEX idx_panel_cache_expires ON panel_cache(expires_at);
`},
```

- [ ] **Step 2: Write failing test**

```go
package storage

import (
	"testing"
	"time"
)

func TestCacheRepo_SetAndGet(t *testing.T) {
	repo, cleanup := testCacheRepo(t)
	defer cleanup()

	err := repo.Set("test:key", `{"hello":"world"}`, 5*time.Minute, "")
	if err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	entry, err := repo.Get("test:key")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if entry.Data != `{"hello":"world"}` {
		t.Errorf("Get() data = %q, want %q", entry.Data, `{"hello":"world"}`)
	}
	if entry.ExpiresAt.Before(time.Now()) {
		t.Error("Get() expired entry returned")
	}
}

func TestCacheRepo_Expired(t *testing.T) {
	repo, cleanup := testCacheRepo(t)
	defer cleanup()

	repo.Set("exp:key", "data", -1*time.Second, "") // expired immediately
	_, err := repo.Get("exp:key")
	if err == nil {
		t.Error("Get() should return error for expired entry")
	}
}

func TestCacheRepo_DeleteByCategory(t *testing.T) {
	repo, cleanup := testCacheRepo(t)
	defer cleanup()

	repo.Set("a:1", "data1", 5*time.Minute, "market")
	repo.Set("a:2", "data2", 5*time.Minute, "market")
	repo.Set("b:1", "data3", 5*time.Minute, "portfolio")

	err := repo.DeleteByCategory("market")
	if err != nil {
		t.Fatalf("DeleteByCategory() error: %v", err)
	}

	if _, err := repo.Get("a:1"); err == nil {
		t.Error("Get('a:1') should error after category delete")
	}
	if _, err := repo.Get("b:1"); err != nil {
		t.Error("Get('b:1') should still exist after category delete")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/storage/ -run TestCacheRepo -v
```
Expected: compilation error (no CacheRepo)

- [ ] **Step 4: Write CacheRepo implementation**

```go
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type CacheEntry struct {
	Key       string    `json:"key"`
	Data      string    `json:"data"`
	ExpiresAt time.Time `json:"expires_at"`
	Category  string    `json:"category,omitempty"`
}

type CacheRepo struct {
	db *sql.DB
}

func NewCacheRepo(db *sql.DB) *CacheRepo {
	return &CacheRepo{db: db}
}

func (r *CacheRepo) Get(key string) (*CacheEntry, error) {
	var entry CacheEntry
	var expiresAt int64
	err := r.db.QueryRow(
		`SELECT key, data, expires_at, COALESCE(category,'') FROM panel_cache WHERE key = ?`, key,
	).Scan(&entry.Key, &entry.Data, &expiresAt, &entry.Category)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cache miss: %s", key)
	}
	if err != nil {
		return nil, err
	}
	entry.ExpiresAt = time.Unix(expiresAt, 0)
	if time.Now().After(entry.ExpiresAt) {
		r.db.Exec(`DELETE FROM panel_cache WHERE key = ?`, key)
		return nil, fmt.Errorf("cache expired: %s", key)
	}
	return &entry, nil
}

func (r *CacheRepo) Set(key, data string, ttl time.Duration, category string) error {
	expiresAt := time.Now().Add(ttl).Unix()
	_, err := r.db.Exec(
		`INSERT INTO panel_cache (key, data, expires_at, category, created_at, updated_at)
		 VALUES (?, ?, ?, ?, unixepoch(), unixepoch())
		 ON CONFLICT(key) DO UPDATE SET data=excluded.data, expires_at=excluded.expires_at,
		   category=excluded.category, updated_at=unixepoch()`,
		key, data, expiresAt, category,
	)
	return err
}

func (r *CacheRepo) Delete(key string) error {
	_, err := r.db.Exec(`DELETE FROM panel_cache WHERE key = ?`, key)
	return err
}

func (r *CacheRepo) DeleteByCategory(category string) error {
	_, err := r.db.Exec(`DELETE FROM panel_cache WHERE category = ?`, category)
	return err
}

func (r *CacheRepo) CleanExpired() error {
	_, err := r.db.Exec(`DELETE FROM panel_cache WHERE expires_at < unixepoch()`)
	return err
}
```

- [ ] **Step 5: Write test helper**

```go
func testCacheRepo(t *testing.T) (*CacheRepo, func()) {
	db, err := sql.Open("sqlite3", ":memory:?_journal_mode=WAL")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS panel_cache (
		key TEXT PRIMARY KEY, data TEXT NOT NULL,
		expires_at INTEGER NOT NULL, category TEXT DEFAULT '',
		created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_panel_cache_category ON panel_cache(category)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_panel_cache_expires ON panel_cache(expires_at)`)
	return NewCacheRepo(db), func() { db.Close() }
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./internal/storage/ -run TestCacheRepo -v
```
Expected: PASS (3/3)

- [ ] **Step 7: Commit**

```bash
git add internal/storage/cache_repo.go internal/storage/cache_repo_test.go
git commit -m "feat(storage): add CacheRepo for offline panel cache (SQLite-backed)"
```

---

### Task 2: Expose cache as Wails-bound Go methods

**Files:**
- Modify: `cmd/app.go` (the Wails App struct with exported Go bindings)

Add two exported Go methods:

```go
// CacheGet retrieves a cached value by key
func (a *App) CacheGet(key string) (string, error) {
    entry, err := a.cacheRepo.Get(key)
    if err != nil {
        return "", err
    }
    return entry.Data, nil
}

// CacheSet stores a value with TTL and optional category
func (a *App) CacheSet(key, data string, ttlSeconds int, category string) error {
    return a.cacheRepo.Set(key, data, time.Duration(ttlSeconds)*time.Second, category)
}

// CacheClear removes cached entries by key prefix or category
func (a *App) CacheClear(keyOrCategory string) error {
    // If contains ':', treat as key; otherwise as category
    if strings.Contains(keyOrCategory, ":") {
        return a.cacheRepo.Delete(keyOrCategory)
    }
    return a.cacheRepo.DeleteByCategory(keyOrCategory)
}
```

- [ ] **Step 1: Add CacheGet/CacheSet/CacheClear methods to App struct**
- [ ] **Step 2: Run backend tests**

```bash
go test ./internal/... -count=1
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: expose CacheGet/CacheSet/CacheClear via Wails IPC"
```

---

### Task 3: Augment frontend fetchWithCache composable with offline fallback

**Files:**
- Modify: `frontend/src/lib/composables/usePanelCache.ts`
- Create: `frontend/src/lib/composables/__tests__/usePanelCache.spec.ts`

**Design:** Replace the in-memory-only cache logic with a 2-tier strategy:

```typescript
import { useWailsApp } from './useWailsApp'

export function usePanelCache() {
  const dataStore = useDataStore()
  const app = useWailsApp()

  // Returns the Wails App method for cache operations (or null if not available)
  function hasPersistentCache(): boolean {
    return !!(app as any)?.CacheGet && !!(app as any)?.CacheSet
  }

  async function fetchWithCache<T>(
    key: string,
    fetcher: () => Promise<T>,
    ttlMs = DEFAULT_TTL,
    category?: string,
  ): Promise<{ data: T; fromCache: boolean; stale?: boolean }> {
    // 1. Try in-memory cache first (fast path)
    const memCached = dataStore.getCached<T>(key)
    if (memCached !== null) {
      return { data: memCached, fromCache: true }
    }

    // 2. Try persistent cache (SQLite) if available
    if (hasPersistentCache()) {
      try {
        const raw = await (app as any).CacheGet(key)
        if (raw) {
          const parsed = JSON.parse(raw) as T
          // Also warm the in-memory cache
          dataStore.setCached(key, parsed, ttlMs)
          return { data: parsed, fromCache: true }
        }
      } catch {
        // Cache miss or expired — fall through to fetcher
      }
    }

    // 3. Fetch from Go backend
    try {
      const data = await fetcher()
      // Warm both caches
      dataStore.setCached(key, data, ttlMs)
      if (hasPersistentCache()) {
        (app as any).CacheSet(key, JSON.stringify(data), Math.ceil(ttlMs / 1000), category || '')
          .catch(() => {}) // fire-and-forget
      }
      return { data, fromCache: false }
    } catch (fetchError) {
      // 4. Offline fallback: try stale data from SQLite
      if (hasPersistentCache()) {
        try {
          const raw = await (app as any).CacheGet(key)
          if (raw) {
            const parsed = JSON.parse(raw) as T
            dataStore.setCached(key, parsed, 0) // keep in memory for this session
            return { data: parsed, fromCache: true, stale: true }
          }
        } catch { /* no stale data either */ }
      }
      throw fetchError // re-throw, PanelShell handles the error state
    }
  }

  return { ... }
}
```

- [ ] **Step 1: Check that CacheGet/CacheSet are added to WailsApp interface in useWailsApp.ts**

Add to `WailsApp` interface:
```typescript
CacheGet(key: string): Promise<string>
CacheSet(key: string, data: string, ttlSeconds: number, category?: string): Promise<void>
CacheClear(keyOrCategory: string): Promise<void>
```

- [ ] **Step 2: Write failing test**

Create `usePanelCache.spec.ts`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock the Wails app
vi.mock('@/lib/composables/useWailsApp', () => ({
  useWailsApp: () => ({
    CacheGet: vi.fn(),
    CacheSet: vi.fn(),
  }),
}))

describe('usePanelCache', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('returns fresh data from fetcher on cache miss', async () => {
    const { fetchWithCache } = await import('../usePanelCache')
    const fetcher = vi.fn().mockResolvedValue('fresh-data')
    const result = await fetchWithCache('test:key', fetcher, 60000)
    expect(result.data).toBe('fresh-data')
    expect(result.fromCache).toBe(false)
  })

  it('returns cached data from in-memory on second call', async () => {
    const { fetchWithCache } = await import('../usePanelCache')
    const fetcher = vi.fn().mockResolvedValue('stored')
    await fetchWithCache('test:key2', fetcher, 60000)
    const second = await fetchWithCache('test:key2', fetcher, 60000)
    expect(second.fromCache).toBe(true)
    expect(second.data).toBe('stored')
    expect(fetcher).toHaveBeenCalledTimes(1)
  })

  it('returns stale data when fetcher fails and persistent cache exists', async () => {
    // Persistent cache mock returns stale data
    const { useWailsApp } = await import('@/lib/composables/useWailsApp')
    const mockApp = useWailsApp()
    mockApp!.CacheGet = vi.fn().mockResolvedValue('"stale-from-sqlite"')

    const { fetchWithCache } = await import('../usePanelCache')
    const fetcher = vi.fn().mockRejectedValue(new Error('network error'))
    const result = await fetchWithCache('stale:key', fetcher, 60000)
    expect(result.data).toBe('stale-from-sqlite')
    expect(result.stale).toBe(true)
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/lib/composables/__tests__/usePanelCache.spec.ts
```
Expected: FAIL

- [ ] **Step 4: Modify usePanelCache.ts** — implement the tiered cache pattern shown above

- [ ] **Step 5: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/lib/composables/__tests__/usePanelCache.spec.ts
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/lib/composables/usePanelCache.ts frontend/src/lib/composables/__tests__/usePanelCache.spec.ts
git commit -m "feat: add offline fallback with persistent cache to fetchWithCache composable"
```

---

### Task 4: Tag high-value panels with cache categories

**Files:**
- Modify: `frontend/src/terminal/panels/WatchlistPanel.vue`
- Modify: `frontend/src/terminal/panels/PortfolioSummary.vue`
- Modify: `frontend/src/terminal/panels/TradeHistory.vue`
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue`
- Modify: `frontend/src/terminal/panels/SettingsPanel.vue`

For each panel, change the `fetchWithCache(key, fetcher)` call to pass a `category` parameter:

Before:
```typescript
const { data } = await fetchWithCache(`quote:${sym}`, () => app.GetQuote(...))
```

After:
```typescript
const { data, stale } = await fetchWithCache(
  `quote:${sym}`,
  () => app.GetQuote(...),
  10 * 1000,
  'watchlist',
)
if (stale) console.warn(`[${sym}] showing stale cache data`)
```

- [ ] **Step 1–5: Update each panel** — add category tag, show stale badge in template
- [ ] **Step 6: Run tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/terminal/
git commit -m "feat(panel): add cache categories to Watchlist, Portfolio, TradeHistory, MarketOverview, Settings"
```

---

### Task 5: Add stale data indicator badge to PanelShell

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelShell.vue`

- [ ] **Step 1: Add `stale` prop and indicator**

```vue
<script setup lang="ts">
defineProps<{
  state: 'loading' | 'loaded' | 'error' | 'empty'
  error?: string
  stale?: boolean   // ← new
}>()
</script>

<template>
  <div class="panel-shell" role="region">
    <div v-if="stale" class="panel-shell-stale-badge" title="数据可能不是最新">
      ⚠ 离线模式
    </div>
    <!-- ... existing content -->
  </div>
</template>

<style scoped>
.panel-shell-stale-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  padding: 2px 8px;
  font-size: 11px;
  color: #b8860b;
  background: #fff8e1;
  border: 1px solid #f0d060;
  border-radius: var(--radius-sm, 4px);
  z-index: 10;
}
</style>
```

- [ ] **Step 2: Run tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelShell.vue
git commit -m "feat(panel): add stale data badge to PanelShell for offline mode"
```

---

### Task 6: Verify full test suite

- [ ] **Step 1: Run Go tests**

```bash
go test ./internal/... -count=1
```
Expected: PASS

- [ ] **Step 2: Run frontend tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 3: Type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: verify offline cache implementation — all tests pass"
```

---

### Task 7: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add entry**

```markdown
### Added
- [Storage] New CacheRepo (SQLite-backed) for persistent panel data cache
- [Frontend] Tiered cache in fetchWithCache composable (in-memory → SQLite → stale fallback)
- [Frontend] Offline stale-data badge indicator on PanelShell
- [Panel] Cache categories for Watchlist, PortfolioSummary, TradeHistory, MarketOverview, Settings
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md && git commit -m "chore: update CHANGELOG for offline cache"
```
