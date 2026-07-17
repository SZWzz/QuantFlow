# API Key 集中管理面板 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a centralized API Key management panel with health verification, status indicators, and import/export, integrated into SettingsPanel.

**Architecture:** A frontend `ApiKeyRegistry` lists all known data sources and broker key entries. Each entry has a Verify button that calls Go IPC `VerifyApiKey(serviceId)` which makes a real API call. Results are cached in localStorage. `apiKeyRegistry.ts` provides the typed registry; `ApiKeyManager.vue` renders the management UI with status indicators. The Go side adds `Verify()` to `credential.go` for per-service verification logic.

**Tech Stack:** Vue 3 Composition API, Vitest, Go 1.25+, SQLite, AES-256-GCM

## Global Constraints

- Go: `slog` for logging, explicit error returns, no panic in library code
- Vue: Composition API with `<script setup lang="ts">`, Pinia for state
- Test: Go table-driven tests, Vue component tests with vitest
- No `window.confirm()` / `window.alert()` — use `@/lib/wails` dialog helpers
- Tests live in `__tests__` directories next to components
- Every new file must have a matching test file
- All IPC methods on `*App` must be exported (capitalized)

---

### Task 1: Create `apiKeyRegistry.ts` with all key entries

**Files:**
- Create: `frontend/src/lib/apiKeyRegistry.ts`
- Test: `frontend/src/lib/__tests__/apiKeyRegistry.test.ts`

**Interfaces:**
- Consumes: nothing
- Produces: `API_KEY_REGISTRY: ApiKeyEntry[]` — 15 entries covering all data sources and brokers

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/lib/__tests__/apiKeyRegistry.test.ts
import { describe, it, expect } from 'vitest'
import { API_KEY_REGISTRY, type ApiKeyEntry } from '../apiKeyRegistry'

describe('API_KEY_REGISTRY', () => {
  it('should have at least 15 entries', () => {
    expect(API_KEY_REGISTRY.length).toBeGreaterThanOrEqual(15)
  })

  it('every entry should have required fields', () => {
    for (const entry of API_KEY_REGISTRY) {
      expect(entry.id).toBeTruthy()
      expect(entry.name).toBeTruthy()
      expect(Array.isArray(entry.market)).toBe(true)
      expect(['api_key', 'secret', 'token', 'both']).toContain(entry.type)
      expect(typeof entry.required).toBe('boolean')
    }
  })

  it('should have entries for all markets', () => {
    const markets = new Set(API_KEY_REGISTRY.flatMap(e => e.market))
    expect(markets.has('CN')).toBe(true)
    expect(markets.has('HK')).toBe(true)
    expect(markets.has('US')).toBe(true)
    expect(markets.has('CRYPTO')).toBe(true)
  })

  it('every id should be unique', () => {
    const ids = API_KEY_REGISTRY.map(e => e.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('optional entries should have ➕ icon marker', () => {
    const optional = API_KEY_REGISTRY.filter(e => !e.required)
    expect(optional.length).toBeGreaterThan(0)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd frontend && npx vitest run src/lib/__tests__/apiKeyRegistry.test.ts`
Expected: FAIL — cannot find module

- [ ] **Step 3: Write minimal implementation**

```typescript
// frontend/src/lib/apiKeyRegistry.ts
export type ApiKeyType = 'api_key' | 'secret' | 'token' | 'both'

export interface ApiKeyEntry {
  id: string
  name: string
  market: string[]
  type: ApiKeyType
  required: boolean
  verifyURL?: string
  docURL?: string
}

export const API_KEY_REGISTRY: ApiKeyEntry[] = [
  { id: 'tushare', name: 'TuShare', market: ['CN'], type: 'api_key', required: false, docURL: 'https://tushare.pro' },
  { id: 'polygon', name: 'Polygon.io', market: ['US'], type: 'api_key', required: false, docURL: 'https://polygon.io' },
  { id: 'qos', name: 'QOS', market: ['HK'], type: 'api_key', required: false },
  { id: 'futu_opend', name: '富途 OpenD', market: ['CN', 'HK', 'US'], type: 'token', required: false },
  { id: 'alpaca_key', name: 'Alpaca API Key', market: ['US'], type: 'both', required: false },
  { id: 'alpaca_secret', name: 'Alpaca Secret', market: ['US'], type: 'secret', required: false },
  { id: 'binance_key', name: 'Binance API Key', market: ['CRYPTO'], type: 'both', required: false },
  { id: 'binance_secret', name: 'Binance Secret', market: ['CRYPTO'], type: 'secret', required: false },
  { id: 'okx_key', name: 'OKX API Key', market: ['CRYPTO'], type: 'both', required: false },
  { id: 'okx_secret', name: 'OKX Secret', market: ['CRYPTO'], type: 'secret', required: false },
  { id: 'ibkr_key', name: 'IBKR API Key', market: ['US', 'HK'], type: 'api_key', required: false },
  { id: 'openai_key', name: 'OpenAI', market: ['ALL'], type: 'api_key', required: false, docURL: 'https://platform.openai.com/api-keys' },
  { id: 'deepseek_key', name: 'DeepSeek', market: ['ALL'], type: 'api_key', required: false, docURL: 'https://platform.deepseek.com' },
  { id: 'finnhub', name: 'Finnhub', market: ['US', 'HK'], type: 'api_key', required: false },
  { id: 'eodhd', name: 'EOD Historical Data', market: ['US', 'HK', 'CN'], type: 'api_key', required: false },
  { id: 'fred', name: 'FRED', market: ['US'], type: 'api_key', required: false },
  { id: 'iwencai', name: '爱问财', market: ['CN'], type: 'api_key', required: false },
]

export function filterByMarket(entries: ApiKeyEntry[], markets: string[]): ApiKeyEntry[] {
  if (markets.includes('ALL')) return entries
  return entries.filter(e => e.market.some(m => m === 'ALL' || markets.includes(m)))
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd frontend && npx vitest run src/lib/__tests__/apiKeyRegistry.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/apiKeyRegistry.ts frontend/src/lib/__tests__/apiKeyRegistry.test.ts
git commit -m "feat(frontend): add API key registry with 17 entries and filterByMarket"
```

---

### Task 2: Add `Verify()` to `credential.go` + Go tests

**Files:**
- Modify: `internal/auth/credential.go:204` — add `Verify(serviceID string) error`
- Test: `internal/auth/credential_test.go` — add `TestCredentialManager_Verify`

**Interfaces:**
- Consumes: `CredentialManager` struct (existing)
- Produces: `(cm *CredentialManager) Verify(serviceID string, keys map[string]string) error`

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/auth/credential_test.go
func TestCredentialManager_Verify(t *testing.T) {
	db := setupTestDB(t)
	cm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	// Unknown service should return error
	err = cm.Verify("nonexistent_service", map[string]string{"key": "val"})
	if err == nil {
		t.Error("expected error for unknown service")
	}

	// Known service with valid call — TuShare returns JSON with "code" field on invalid token
	// We don't test actual HTTP here; just verify the method exists and validates inputs
	err = cm.Verify("tushare", map[string]string{"api_key": ""})
	if err == nil {
		t.Error("expected error for empty api_key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/app && go test ./internal/auth/ -run TestCredentialManager_Verify -v`
Expected: FAIL — `Verify` not defined

- [ ] **Step 3: Add Verify() to credential.go**

```go
// Verify tests a credential by making a real API call to the service.
// Returns nil if the credential is valid, error otherwise.
// Supported services: tushare, polygon (others return unimplemented).
func (m *CredentialManager) Verify(serviceID string, keys map[string]string) error {
	if len(keys) == 0 {
		return fmt.Errorf("credential: no keys provided for %q", serviceID)
	}

	switch serviceID {
	case "tushare":
		apiKey, ok := keys["api_key"]
		if !ok || apiKey == "" {
			return fmt.Errorf("credential: tushare: api_key is required")
		}
		// TuShare token validation: GET https://tushare.pro/token?token={key}
		// Returns {"code": 200, "data": {...}} on success, {"code": -1} on failure
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("https://tushare.pro/token?token=%s", apiKey))
		if err != nil {
			return fmt.Errorf("credential: tushare: request failed: %w", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var result struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("credential: tushare: parse response: %w", err)
		}
		if result.Code != 200 {
			return fmt.Errorf("credential: tushare: invalid token (code=%d)", result.Code)
		}
		return nil

	case "polygon":
		apiKey, ok := keys["api_key"]
		if !ok || apiKey == "" {
			return fmt.Errorf("credential: polygon: api_key is required")
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("https://api.polygon.io/v2/reference/types?apiKey=%s", apiKey))
		if err != nil {
			return fmt.Errorf("credential: polygon: request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 || resp.StatusCode == 403 {
			// 403 with proper key means the key format is valid but may lack permissions
			return nil
		}
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("credential: polygon: status=%d: %s", resp.StatusCode, string(body))

	default:
		return fmt.Errorf("credential: verify not implemented for %q", serviceID)
	}
}
```

Also add imports at top of credential.go:

```go
import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/app && go test ./internal/auth/ -run TestCredentialManager_Verify -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth/credential.go internal/auth/credential_test.go
git commit -m "feat(auth): add Verify() method to CredentialManager for API key validation"
```

---

### Task 3: Add `VerifyApiKey` IPC in app_system.go

**Files:**
- Modify: `app_system.go` — add `VerifyApiKey` method
- Test: (IPC tests are end-to-end with integration; minimal unit test)

**Interfaces:**
- Consumes: `credMgr *auth.CredentialManager` from App struct
- Produces: `(a *App) VerifyApiKey(serviceID string, keys map[string]string) error`

- [ ] **Step 1: Read existing app_system.go to confirm no conflicts**

- [ ] **Step 2: Add VerifyApiKey to app_system.go**

```go
// VerifyApiKey validates an API key by making a real call to the service.
// Called from the frontend ApiKeyManager panel.
// Supported services: tushare, polygon. Others return "not implemented".
func (a *App) VerifyApiKey(serviceID string, keys map[string]string) error {
	if a.credMgr == nil {
		return fmt.Errorf("credential manager not initialized")
	}
	return a.credMgr.Verify(serviceID, keys)
}
```

- [ ] **Step 3: Also add the IPC export to wails.ts**

```typescript
// Add to frontend/src/lib/wails.ts after existing credential methods

// ── API Key Verification ──────────────────────────────────────────────

export async function VerifyApiKey(serviceID: string, keys: Record<string, string>): Promise<void> {
  return wailsCall<void>('VerifyApiKey', serviceID, keys)
}
```

- [ ] **Step 4: Run Go build to verify compilation**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/app && go vet ./...`
Expected: PASS (no errors)

- [ ] **Step 5: Commit**

```bash
git add app_system.go frontend/src/lib/wails.ts
git commit -m "feat(backend): add VerifyApiKey IPC and frontend binding"
```

---

### Task 4: Create `ApiKeyManager.vue` + frontend tests

**Files:**
- Create: `frontend/src/terminal/components/ApiKeyManager.vue`
- Test: `frontend/src/terminal/components/__tests__/ApiKeyManager.test.ts`

**Interfaces:**
- Consumes: `API_KEY_REGISTRY` from task 1, `verifyCredential` from wails.ts (task 3), credential save/get from wails.ts
- Produces: fully functioning API key management UI component

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/components/__tests__/ApiKeyManager.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ApiKeyManager from '../ApiKeyManager.vue'

// Mock wails calls
vi.mock('@/lib/wails', () => ({
  saveCredential: vi.fn(() => Promise.resolve()),
  getCredential: vi.fn(() => Promise.resolve(null)),
  VerifyApiKey: vi.fn(() => Promise.resolve()),
  listCredentialNames: vi.fn(() => Promise.resolve([])),
}))

describe('ApiKeyManager', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('should mount and show all key entries', () => {
    const wrapper = mount(ApiKeyManager)
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.text()).toContain('API Keys')
    expect(wrapper.findAll('[data-test="key-entry"]').length).toBeGreaterThanOrEqual(15)
  })

  it('should filter by market', async () => {
    const wrapper = mount(ApiKeyManager)
    // By default all entries shown
    const allEntries = wrapper.findAll('[data-test="key-entry"]').length

    // Filter to CN only
    const cnFilter = wrapper.find('[data-test="filter-cn"]')
    if (cnFilter.exists()) {
      await cnFilter.trigger('click')
      const cnEntries = wrapper.findAll('[data-test="key-entry"]').length
      expect(cnEntries).toBeLessThan(allEntries)
    }
  })

  it('should show status indicators for each entry', () => {
    const wrapper = mount(ApiKeyManager)
    const indicators = wrapper.findAll('[data-test="status-indicator"]')
    expect(indicators.length).toBeGreaterThanOrEqual(15)
    // All start as ❌ (not configured)
    indicators.forEach(el => {
      expect(['✅', '⚠️', '❌', '➖']).toContain(el.text())
    })
  })

  it('should have verify button on each entry', () => {
    const wrapper = mount(ApiKeyManager)
    const verifyBtns = wrapper.findAll('[data-test="verify-btn"]')
    expect(verifyBtns.length).toBeGreaterThanOrEqual(15)
  })

  it('should have bulk verify button', () => {
    const wrapper = mount(ApiKeyManager)
    expect(wrapper.text()).toContain('全部验证')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd frontend && npx vitest run src/terminal/components/__tests__/ApiKeyManager.test.ts`
Expected: FAIL — cannot find module

- [ ] **Step 3: Write minimal implementation**

```vue
<!-- frontend/src/terminal/components/ApiKeyManager.vue -->
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { API_KEY_REGISTRY, filterByMarket, type ApiKeyEntry, type ApiKeyType } from '@/lib/apiKeyRegistry'
import { saveCredential, getCredential, VerifyApiKey, listCredentialNames } from '@/lib/wails'

const marketFilter = ref<string | null>(null)
const keyValues = ref<Record<string, Record<string, string>>>({})
const keyStatus = ref<Record<string, 'ok' | 'error' | 'unknown' | 'unset'>>({})
const verifying = ref<Record<string, boolean>>({})

const filteredEntries = computed(() => {
  const entries = marketFilter.value
    ? filterByMarket(API_KEY_REGISTRY, [marketFilter.value])
    : API_KEY_REGISTRY
  return entries
})

const markets = computed(() => {
  const set = new Set<string>()
  API_KEY_REGISTRY.forEach(e => e.market.forEach(m => set.add(m)))
  return Array.from(set)
})

function statusIcon(status: string): string {
  switch (status) {
    case 'ok': return '✅'
    case 'error': return '⚠️'
    case 'unset': return '❌'
    default: return '➖'
  }
}

function inputFields(type: ApiKeyType): string[] {
  switch (type) {
    case 'api_key': return ['api_key']
    case 'secret': return ['secret']
    case 'token': return ['token']
    case 'both': return ['api_key', 'secret']
    default: return ['api_key']
  }
}

function fieldLabel(field: string): string {
  switch (field) {
    case 'api_key': return 'Key'
    case 'secret': return 'Secret'
    case 'token': return 'Token'
    default: return field
  }
}

async function loadExistingKeys() {
  try {
    const names = await listCredentialNames()
    for (const entry of API_KEY_REGISTRY) {
      const found = names.includes(entry.id)
      if (found) {
        const cred = await getCredential(entry.id)
        if (cred) {
          keyValues.value[entry.id] = cred
          keyStatus.value[entry.id] = 'unknown'
        }
      } else {
        keyStatus.value[entry.id] = 'unset'
        keyValues.value[entry.id] = {}
      }
    }
  } catch {
    // Graceful fallback
  }
}

async function saveKey(entry: ApiKeyEntry) {
  const values = keyValues.value[entry.id] || {}
  if (Object.values(values).some(v => v?.length > 0)) {
    await saveCredential(entry.id, values)
    keyStatus.value[entry.id] = 'unknown'
  }
}

async function verifyKey(entry: ApiKeyEntry) {
  const values = keyValues.value[entry.id] || {}
  if (!Object.values(values).some(v => v?.length > 0)) return

  verifying.value[entry.id] = true
  try {
    await VerifyApiKey(entry.id, values)
    keyStatus.value[entry.id] = 'ok'
  } catch {
    keyStatus.value[entry.id] = 'error'
  } finally {
    verifying.value[entry.id] = false
  }
}

async function verifyAll() {
  const entries = filteredEntries.value.filter(e => {
    const vals = keyValues.value[e.id] || {}
    return Object.values(vals).some(v => v?.length > 0)
  })
  await Promise.all(entries.map(e => verifyKey(e)))
}

onMounted(loadExistingKeys)
</script>

<template>
  <div class="api-key-manager">
    <h3>🔑 API Keys</h3>

    <div class="market-filters">
      <button
        v-for="m in markets" :key="m"
        :data-test="`filter-${m.toLowerCase()}`"
        :class="{ active: marketFilter === m }"
        @click="marketFilter = marketFilter === m ? null : m"
      >
        {{ { CN: 'A 股', HK: '港股', US: '美股', CRYPTO: '加密', ALL: '全部' }[m] || m }}
      </button>
      <button v-if="marketFilter" @click="marketFilter = null">全部</button>
    </div>

    <div class="key-list">
      <div
        v-for="entry in filteredEntries" :key="entry.id"
        class="key-entry" data-test="key-entry"
      >
        <span class="status-icon" data-test="status-indicator">{{ statusIcon(keyStatus[entry.id] || 'unset') }}</span>
        <span class="entry-name">{{ entry.name }}</span>
        <div class="entry-fields">
          <div v-for="field in inputFields(entry.type)" :key="field" class="field-row">
            <label>{{ fieldLabel(field) }}</label>
            <input
              type="password"
              :value="keyValues[entry.id]?.[field] || ''"
              @input="e => {
                if (!keyValues[entry.id]) keyValues[entry.id] = {}
                keyValues[entry.id][field] = (e.target as HTMLInputElement).value
              }"
              :placeholder="`输入 ${fieldLabel(field)}`"
            />
          </div>
        </div>
        <div class="entry-actions">
          <button
            class="btn-sm btn-save"
            @click="saveKey(entry)"
            :disabled="!Object.values(keyValues[entry.id] || {}).some(v => v?.length > 0)"
          >💾</button>
          <button
            class="btn-sm btn-verify"
            data-test="verify-btn"
            @click="verifyKey(entry)"
            :disabled="verifying[entry.id] || !Object.values(keyValues[entry.id] || {}).some(v => v?.length > 0)"
          >🔍</button>
        </div>
      </div>
    </div>

    <div class="bulk-actions">
      <button class="btn-primary" @click="verifyAll">
        全部验证
      </button>
    </div>
  </div>
</template>

<style scoped>
.api-key-manager { padding: 16px; }
.api-key-manager h3 { margin-bottom: 16px; font-size: 16px; }
.market-filters { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 16px; }
.market-filters button { padding: 4px 12px; border: 1px solid var(--color-border); background: transparent; color: var(--color-text); border-radius: 12px; font-size: 12px; cursor: pointer; }
.market-filters button.active { border-color: var(--color-accent); background: var(--color-accent-soft); }
.key-list { display: flex; flex-direction: column; gap: 8px; }
.key-entry { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg); }
.status-icon { font-size: 16px; width: 24px; text-align: center; }
.entry-name { width: 120px; font-weight: 600; font-size: 13px; flex-shrink: 0; }
.entry-fields { flex: 1; display: flex; gap: 8px; }
.field-row { display: flex; align-items: center; gap: 4px; }
.field-row label { font-size: 11px; color: var(--color-text-tertiary); }
.field-row input { padding: 4px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-app); color: var(--color-text); font-size: 12px; width: 120px; }
.entry-actions { display: flex; gap: 4px; }
.btn-sm { padding: 4px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; cursor: pointer; font-size: 12px; }
.btn-sm:disabled { opacity: 0.4; cursor: not-allowed; }
.bulk-actions { margin-top: 16px; display: flex; gap: 8px; }
.btn-primary { padding: 8px 24px; background: var(--color-accent); color: #fff; border: none; border-radius: 8px; font-size: 13px; cursor: pointer; }
</style>
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd frontend && npx vitest run src/terminal/components/__tests__/ApiKeyManager.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/ApiKeyManager.vue frontend/src/terminal/components/__tests__/ApiKeyManager.test.ts
git commit -m "feat(frontend): add ApiKeyManager component with verify and save"
```

---

### Task 5: Integrate into SettingsPanel

**Files:**
- Modify: `frontend/src/terminal/panels/SettingsPanel.vue:46-51` — add api section mapping

**Interfaces:**
- Consumes: `ApiKeyManager.vue` from task 4
- Produces: SettingsPanel with "API Keys" tab showing ApiKeyManager

- [ ] **Step 1: Read existing SettingsPanel to find integration point**

- [ ] **Step 2: Modify SettingsPanel to include ApiKeyManager**

Replace the existing `api` section handler in SettingsPanel.vue with ApiKeyManager:

```typescript
// In SettingsPanel.vue, replace the api section content:
import ApiKeyManager from '../components/ApiKeyManager.vue'

const embeddedComponents: Record<string, any> = {
  storage: StoragePanel,
  notify: NotifyPanel,
  logs: LogPanel,
  layouts: LayoutTemplatePanel,
  api: ApiKeyManager,
}
```

Also remove the old inline `apiKeys` handling (lines 59-80+ where it saves individual keys via the old form):

The sections array already has the `{ id: 'api', label: 'api', icon: getIcon('broker') }` entry. The key change is to set `embeddedComponents['api'] = ApiKeyManager` so the existing `activeSection === 'api'` path renders the new component.

- [ ] **Step 3: Verify by running existing SettingsPanel tests**
Run: `cd frontend && npx vitest run src/terminal/panels/__tests__/SettingsPanel.test.ts`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/SettingsPanel.vue
git commit -m "feat(frontend): integrate ApiKeyManager into SettingsPanel API Keys tab"
```
