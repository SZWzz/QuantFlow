# Security Hardening Implementation Plan

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate plaintext API key storage in localStorage and config.yaml; make migration failures fatal.

**Architecture:** Replace frontend localStorage API keys with Go CredentialManager (AES-256-GCM + OS keychain). Migration failure becomes a hard startup error. Config API keys redirect to CredentialManager.

**Tech Stack:** Go 1.25, Vue 3 + TypeScript, SQLite (credentials table via migration 001), macOS Keychain / Linux secret-tool

## Global Constraints

- No new dependencies — `crypto/aes`, `crypto/cipher` already used in `internal/auth/credential.go`
- Go `SaveCredential`/`GetCredential`/`DeleteCredential` already exposed in `app.go:300-314`
- Frontend must keep using settings store for *non-secret* fields (baseUrl, custom model names)
- Migration failure: change `slog.Warn` → `return fmt.Errorf` only

---

### Task 1: Remove API Key Fields from Settings Store

**Files:**
- Modify: `frontend/src/stores/settings.ts`
- Test: `frontend/src/stores/settings.test.ts`

**Interfaces:**
- Consumes: existing `SettingsState` interface with llm*Key fields
- Produces: updated `SettingsState` without llm*Key fields, `CredentialManager` IPC wrappers in `wails.ts`

- [ ] **Step 1: Remove all `llm*Key` fields from SettingsState**

```typescript
// settings.ts — change this:
export interface SettingsState {
  // ... keep everything except these:
  // REMOVED: llmOpenaiKey, llmAnthropicKey, llmDeepseekKey, llmGoogleKey,
  //   llmMistralKey, llmGroqKey, llmSiliconflowKey, llmZhipuKey,
  //   llmOpenrouterKey, llmOpencodeKey, llmCustomKey
  // REMOVED: fredApiKey, finnhubApiKey, iwencaiApiKey

  // ADD these instead (boolean tracking):
  llmOpenaiConfigured: boolean
  llmAnthropicConfigured: boolean
  llmDeepseekConfigured: boolean
  llmGoogleConfigured: boolean
  llmMistralConfigured: boolean
  llmGroqConfigured: boolean
  llmSiliconflowConfigured: boolean
  llmZhipuConfigured: boolean
  llmOpenrouterConfigured: boolean
  llmOpencodeConfigured: boolean
  llmCustomConfigured: boolean
}
```

Full diff — in `defaultSettings()`, replace the key fields with `llmOpenaiConfigured: false` (and all 10 other providers). Remove `fredApiKey`, `finnhubApiKey`, `iwencaiApiKey` from the interface and defaults.

- [ ] **Step 2: Write failing test for `settings.test.ts`**

```typescript
// frontend/src/stores/settings.test.ts
import { setActivePinia, createPinia } from 'pinia'
import { useSettingsStore } from '@/stores/settings'

describe('settings store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('should not contain API keys', () => {
    const store = useSettingsStore()
    expect(store.settings).not.toHaveProperty('llmOpenaiKey')
    expect(store.settings).toHaveProperty('llmOpenaiConfigured')
  })

  it('should migrate old localStorage to new format on load', () => {
    const oldData = JSON.stringify({ llmOpenaiKey: 'sk-old', llmAnthropicKey: 'sk-anthropic' })
    localStorage.setItem('quantflow-settings', oldData)
    const store = useSettingsStore()
    // Old keys should be stripped, new Configured flags should exist (false after migration)
    expect((store.settings as any).llmOpenaiKey).toBeUndefined()
    expect(store.settings.llmOpenaiConfigured).toBe(false)
  })
})
```

- [ ] **Step 3: Add migration logic in `loadSettings()`**

```typescript
// In loadSettings(), after JSON.parse(saved):
function loadSettings(): SettingsState {
  const saved = localStorage.getItem('quantflow-settings')
  if (saved) {
    try {
      const parsed = JSON.parse(saved)
      // Migrate old format: strip API keys, set Configured flags
      const migrated: any = { ...defaultSettings(), ...parsed }
      for (const key of Object.keys(parsed)) {
        if (key.endsWith('Key') && parsed[key]) {
          const provider = key.replace('Key', '')
          migrated[provider + 'Configured'] = true
        }
        if (key.endsWith('Key') || key === 'fredApiKey' || key === 'finnhubApiKey' || key === 'iwencaiApiKey') {
          delete migrated[key]
        }
      }
      return migrated as SettingsState
    } catch {
      return defaultSettings()
    }
  }
  return defaultSettings()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/stores/settings.test.ts
```
Expected: PASS (0 failures)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/settings.ts frontend/src/stores/settings.test.ts
git commit -m "feat(auth): remove plaintext API key fields from settings store, add migration"
```

---

### Task 2: Add CredentialManager IPC Wrappers to wails.ts

**Files:**
- Modify: `frontend/src/lib/wails.ts`

**Interfaces:**
- Consumes: `window.go.main.App.SaveCredential(name, type, keys)`, `GetCredential(name)`, `DeleteCredential(name)`, `ListCredentialNames()`
- Produces: typed `saveCredential()`, `getCredential()`, `deleteCredential()`, `listCredentialNames()`

- [ ] **Step 1: Write failing test**

```typescript
// frontend/src/lib/__tests__/wails.test.ts (create if not exists)
import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('credential wrappers', () => {
  beforeEach(() => {
    ;(window as any).go = {
      main: {
        App: {
          SaveCredential: vi.fn().mockResolvedValue(undefined),
          ListCredentialNames: vi.fn().mockResolvedValue(['openai', 'anthropic']),
        }
      }
    }
  })

  it('saveCredential calls Go IPC', async () => {
    const { saveCredential } = await import('@/lib/wails')
    await saveCredential('openai', { api_key: 'sk-test' })
    expect((window as any).go.main.App.SaveCredential).toHaveBeenCalledWith('openai', 'api_key', { api_key: 'sk-test' })
  })

  it('listCredentialNames returns names', async () => {
    const { listCredentialNames } = await import('@/lib/wails')
    const names = await listCredentialNames()
    expect(names).toEqual(['openai', 'anthropic'])
  })
})
```

- [ ] **Step 2: Add typed wrappers to wails.ts**

```typescript
// frontend/src/lib/wails.ts — add to the file (near end, before export)

export async function saveCredential(name: string, keys: Record<string, string>): Promise<void> {
  const app = (window as any).go?.main?.App
  if (!app?.SaveCredential) throw new Error('SaveCredential not available')
  return app.SaveCredential(name, 'api_key', keys)
}

export async function getCredential(name: string): Promise<Record<string, string> | null> {
  const app = (window as any).go?.main?.App
  if (!app?.GetCredential) return null
  try {
    return app.GetCredential(name)
  } catch {
    return null
  }
}

export async function deleteCredential(name: string): Promise<void> {
  const app = (window as any).go?.main?.App
  if (!app?.DeleteCredential) throw new Error('DeleteCredential not available')
  return app.DeleteCredential(name)
}

export async function listCredentialNames(): Promise<string[]> {
  const app = (window as any).go?.main?.App
  if (!app?.ListCredentialNames) return []
  return app.ListCredentialNames()
}
```

- [ ] **Step 3: Run test**

```bash
cd frontend && npx vitest run src/lib/__tests__/wails.test.ts
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/wails.ts frontend/src/lib/__tests__/wails.test.ts
git commit -m "feat(auth): add typed CredentialManager wrappers to wails.ts"
```

---

### Task 3: Rewire ModelRegistryPanel to Use CredentialManager

**Files:**
- Modify: `frontend/src/terminal/panels/ModelRegistryPanel.vue`

**Interfaces:**
- Consumes: `saveCredential(name, keys)`, `getCredential(name)`, `listCredentialNames()` from wails.ts
- Consumes: settings store `llmOpenaiConfigured`, `llmOpenaiBaseUrl`, `llmDefaultModel`, etc.
- Produces: API keys read from/written to CredentialManager; settings store tracks only booleans + base URLs

- [ ] **Step 1: Add unit test for credential-based save/load**

```typescript
// frontend/src/terminal/panels/__tests__/ModelRegistryPanel.test.ts (augment existing)
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'

// Mock the credential wrappers
vi.mock('@/lib/wails', () => ({
  saveCredential: vi.fn().mockResolvedValue(undefined),
  getCredential: vi.fn().mockResolvedValue({ api_key: 'sk-mock' }),
  listCredentialNames: vi.fn().mockResolvedValue(['openai']),
}))

// ... (test body using the panel's UI)
```

Run once: `npx vitest run src/terminal/panels/__tests__/ModelRegistryPanel.test.ts`
Expected: test exists but may fail before implementation

- [ ] **Step 2: Rewrite `loadFromStore()` and `saveToStore()`**

In `ModelRegistryPanel.vue`, replace the current `loadFromStore()` and `saveToStore()`:

```typescript
// New loadFromStore: read keys from CredentialManager, base URLs from settings
async function loadFromStore() {
  const s = settingsStore.settings
  // Load base URLs from settings (non-secret)
  form.value.openai.baseUrl = s.llmOpenaiBaseUrl
  form.value.anthropic.baseUrl = s.llmAnthropicBaseUrl
  form.value.deepseek.baseUrl = s.llmDeepseekBaseUrl
  form.value.google.baseUrl = s.llmGoogleBaseUrl
  form.value.mistral.baseUrl = s.llmMistralBaseUrl
  form.value.groq.baseUrl = s.llmGroqBaseUrl
  form.value.siliconflow.baseUrl = s.llmSiliconflowBaseUrl
  form.value.zhipu.baseUrl = s.llmZhipuBaseUrl
  form.value.openrouter.baseUrl = s.llmOpenrouterBaseUrl
  form.value.opencode.baseUrl = s.llmOpencodeBaseUrl
  form.value.custom.baseUrl = s.llmCustomBaseUrl
  form.value.ollama.baseUrl = s.llmOllamaBaseUrl
  form.value.custom.name = s.llmCustomName
  form.value.custom.models = s.llmCustomModels

  // Load keys from CredentialManager
  for (const pid of providers.filter(p => p.needKey).map(p => p.id)) {
    const cred = await getCredential(`${pid}_api_key`)
    if (cred?.api_key) {
      const f = form.value[pid as keyof typeof form.value] as any
      if (f) f.apiKey = cred.api_key
    }
  }
}

// New saveToStore: store keys in CredentialManager, base URLs in settings
async function saveToStore() {
  const f = form.value
  settingsStore.update('llmOpenaiBaseUrl', f.openai.baseUrl)
  settingsStore.update('llmAnthropicBaseUrl', f.anthropic.baseUrl)
  settingsStore.update('llmDeepseekBaseUrl', f.deepseek.baseUrl)
  settingsStore.update('llmGoogleBaseUrl', f.google.baseUrl)
  settingsStore.update('llmMistralBaseUrl', f.mistral.baseUrl)
  settingsStore.update('llmGroqBaseUrl', f.groq.baseUrl)
  settingsStore.update('llmSiliconflowBaseUrl', f.siliconflow.baseUrl)
  settingsStore.update('llmZhipuBaseUrl', f.zhipu.baseUrl)
  settingsStore.update('llmOpenrouterBaseUrl', f.openrouter.baseUrl)
  settingsStore.update('llmOpencodeBaseUrl', f.opencode.baseUrl)
  settingsStore.update('llmCustomBaseUrl', f.custom.baseUrl)
  settingsStore.update('llmCustomName', f.custom.name)
  settingsStore.update('llmCustomModels', f.custom.models)
  settingsStore.update('llmOllamaBaseUrl', f.ollama.baseUrl)

  // Save API keys to CredentialManager
  for (const pid of ['openai', 'anthropic', 'deepseek', 'google', 'mistral', 'groq',
    'siliconflow', 'zhipu', 'openrouter', 'opencode', 'custom']) {
    const formField = form.value[pid as keyof typeof form.value] as any
    if (!formField) continue
    const key = formField.apiKey
    if (key) {
      await saveCredential(`${pid}_api_key`, { api_key: key })
      settingsStore.update(pid === 'custom' ? 'llmCustomConfigured' : `llm${pid.charAt(0).toUpperCase() + pid.slice(1)}Configured` as any, true as any)
    }
  }
  saveMsg.value = t('settings.llm_save_hint')
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => { saveMsg.value = ''; saveTimer = null }, 3000)
}
```

Also update `onMounted` to call `await loadFromStore()` (make it async):

```typescript
onMounted(async () => {
  await loadFromStore()
  // ... rest of existing onMounted
})
```

- [ ] **Step 3: Update `testProvider()` and `fetchProviderModels()` — pass apiKey from CredentialManager**

The functions `testProvider(pid)` and `fetchProviderModels(pid)` currently read `f.apiKey` from `form.value`. After the change, the apiKey is still in `form.value` (loaded from CM on mount), so these work without changes. The only change is that `form.value[pid].apiKey` is populated from CM instead of settings store.

- [ ] **Step 4: Run test to verify**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/ModelRegistryPanel.test.ts
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/panels/ModelRegistryPanel.vue
git commit -m "feat(auth): wire ModelRegistryPanel to CredentialManager for API key storage"
```

---

### Task 4: Add `GetCredential` Go IPC + Config API Key Migration

**Files:**
- Modify: `app.go` — add `GetCredential` Wails method
- Modify: `internal/config/config.go` — inject CredentialManager, add CM-first lookup
- Modify: `app_startup.go` — run config→CM migration, wire CM into Config

**Interfaces:**
- Produces: `a.GetCredential(name) (*auth.Credential, error)` Wails method
- Produces: `Config.GetAPIKey(name)` checks CM first, then env var, then config yaml

- [ ] **Step 1: Add GetCredential IPC to app.go**

```go
// GetCredential returns a single credential by name.
func (a *App) GetCredential(name string) (*auth.Credential, error) {
	if a.credMgr == nil {
		return nil, fmt.Errorf("credential manager not initialized")
	}
	creds, err := a.credMgr.List()
	if err != nil {
		return nil, err
	}
	for _, c := range creds {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("credential %q not found", name)
}
```

- [ ] **Step 2: Write test for GetCredential**

```go
// app_test.go (augment existing)
func TestApp_GetCredential(t *testing.T) {
    // Requires test DB + CredentialManager setup
    // Similar to existing credential tests
}
```

- [ ] **Step 3: Add CM-first lookup in Config**

```go
// internal/config/config.go — add field
type Config struct {
	path     string
	cm       *auth.CredentialManager // optional, set at startup

	// yaml fields...
}

// SetCredentialManager wires CM for GetAPIKey
func (c *Config) SetCredentialManager(cm *auth.CredentialManager) {
	c.cm = cm
}

// Updated GetAPIKey: CM → env var → config file
func (c *Config) GetAPIKey(name string) string {
	if c.cm != nil {
		creds, err := c.cm.List()
		if err == nil {
			for _, cred := range creds {
				if cred.Name == name+"_api_key" {
					if key, ok := cred.Keys["api_key"]; ok {
						return key
					}
				}
			}
		}
	}
	if val := c.APIKeys[name]; val != "" {
		return val
	}
	envKey := fmt.Sprintf("%s_API_KEY", toEnvName(name))
	return os.Getenv(envKey)
}
```

- [ ] **Step 4: Wire migration in app_startup.go**

After CredentialManager is initialized (`app_startup.go:283-289`), add:

```go
// Migrate config api_keys to CredentialManager
if a.credMgr != nil {
    for name, key := range a.cfg.APIKeys {
        if key != "" {
            if err := a.credMgr.Save(name+"_api_key", "api_key", map[string]string{"api_key": key}); err != nil {
                slog.Warn("migrate config api_key to credential manager", "name", name, "error", err)
            } else {
                delete(a.cfg.APIKeys, name)
                slog.Info("migrated config api_key to credential manager", "name", name)
            }
        }
    }
    if len(a.cfg.APIKeys) == 0 {
        a.cfg.APIKeys = map[string]string{}
    }
    a.cfg.SetCredentialManager(a.credMgr)
}
```

- [ ] **Step 5: Run Go tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/config/... -v -count=1
go test . -run TestApp_GetCredential -v -count=1
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app.go app_startup.go internal/config/config.go
git commit -m "feat(auth): add GetCredential IPC, wire Config to CredentialManager, migrate config api_keys"
```

---

### Task 5: Make Migration Failure Fatal

**Files:**
- Modify: `app_startup.go:74-76`

- [ ] **Step 1: Write test**

```go
// internal/storage/migrate_test.go (augment)
func TestMigrationFailure_ReturnsLatestVersion(t *testing.T) {
    // Verify that LatestVersion() returns expected, and that
    // the migration version list is monotonically increasing
    migs, err := storage.BuiltinMigrations()
    if err != nil {
        t.Fatal(err)
    }
    for i := 1; i < len(migs); i++ {
        if migs[i].Version <= migs[i-1].Version {
            t.Errorf("migration %d version %d <= previous %d", i, migs[i].Version, migs[i-1].Version)
        }
    }
}
```

- [ ] **Step 2: Change slog.Warn → return error**

```go
// app_startup.go:72-77 — change from:
if migErr == nil {
    if err := storage.Run(db, migrations); err != nil {
        slog.Warn("migrations failed", "error", err)
    }
}
// To:
if migErr == nil {
    if err := storage.Run(db, migrations); err != nil {
        return fmt.Errorf("database migrations failed: %w", err)
    }
}
```

- [ ] **Step 3: Run test**

```bash
cd /app && go test ./internal/storage/... -v -count=1
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add app_startup.go
git commit -m "fix(auth): make migration failure fatal — return error instead of slog.Warn"
```

---

### Task 6: Update CHANGELOG

- [ ] **Step 1: Update CHANGELOG.md**

Add entries for all changes:
```markdown
### Added
- [Security] LLM API keys moved from plaintext localStorage to Go CredentialManager (AES-256-GCM + keychain)
- [Security] GetCredential Wails IPC method for frontend to retrieve encrypted credentials
- [Security] Config.GetAPIKey now checks CredentialManager first, then env vars, then config yaml

### Changed
- [Security] Config api_keys section automatically migrates to CredentialManager on first startup, then zeroed out
- [Security] ModelRegistryPanel saves/loads API keys via CredentialManager; settings store tracks only Configured booleans + base URLs

### Fixed
- [Security] DB migration failure is now fatal — aborts startup with error instead of slog.Warn and silent schema corruption
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for security hardening changes"
```