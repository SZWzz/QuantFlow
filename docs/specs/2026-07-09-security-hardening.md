# Security Hardening: API Keys, Credential Storage, Migration Safety

## Motivation

Phase 12 review identified 3 P0 security/data-integrity issues:

1. **15+ LLM API keys stored in plaintext in `localStorage`** (`frontend/src/stores/settings.ts`). Any XSS or browser extension can exfiltrate all AI provider credentials.

2. **Config file API keys in plaintext** (`internal/config/config.go:18`). The `config.yaml` `api_keys` map serializes keys as-is. The project already has AES-256-GCM + macOS Keychain via `CredentialManager` — but config persistence bypasses it entirely.

3. **Migration failure is non-fatal** (`app_startup.go:75`). A failed schema migration logs `slog.Warn` and continues. If the schema doesn't match code expectations, data can be silently corrupted.

All three can cause real harm: credential theft, financial data corruption, or both.

## Design

### 1a. Frontend API Keys → Go CredentialManager

**Current flow:**
```
settings.ts: localStorage.setItem('quantflow-settings', JSON.stringify({..., llmOpenaiKey: 'sk-...', ...}))
```

**Target flow:**
```
User enters API key in ModelRegistryPanel/SettingPanel
  → Go IPC: SaveCredential(provider+"_api_key", key)   [AES-256-GCM + Keychain]
  → Frontend only stores a boolean "is_configured" in localStorage
  → On use: Go IPC: GetCredential(provider+"_api_key")  [decrypt on read]
  → CredentialManager already exists at internal/auth/credential.go
```

**Modified files:**
- `frontend/src/stores/settings.ts` — Remove all `llm*Key` fields; replace with `llm*Configured: boolean`
- `frontend/src/terminal/panels/ModelRegistryPanel.vue` — Wire API key input to `SaveCredential`/`GetCredential` instead of settings store
- `frontend/src/lib/wails.ts` — Add `saveCredential`/`getCredential`/`deleteCredential` typed wrappers
- `app.go` — Expose `SaveCredential`/`GetCredential`/`DeleteCredential` Wails methods (may already exist; verify)
- `internal/auth/credential.go` — Add `SaveCredential(service, key string) error` and `GetCredential(service string) (string, error)` if not present (currently uses `Save(key, value)` / `List()` pattern)

### 1b. Config API Keys → CredentialManager

**Current flow:**
```
config.yaml:
  api_keys:
    fred: ABC123
    finnhub: DEF456
Config.GetAPIKey("fred") → reads from yaml or env var
```

**Target flow:**
```
Config.GetAPIKey(name) → first tries CredentialManager, then env var, then config file
Add a one-time migration: read all api_keys from config.yaml, write to CredentialManager, zero out config.yaml
```

**Modified files:**
- `internal/config/config.go` — Add `cm *auth.CredentialManager` field; `GetAPIKey` checks CM first
- `app_startup.go` — Inject CredentialManager into Config after init; trigger migration
- `internal/auth/credential.go` — Ensure `Save` works with the same key names used by config

### 1c. Migration Failure → Fatal

**Current:**
```go
if err := migrate.Run(db); err != nil {
    slog.Warn("migrations failed", "error", err) // continues!
}
```

**Target:**
```go
if err := migrate.Run(db); err != nil {
    return fmt.Errorf("migrations failed: %w", err)
}
```

Also add a startup guard: if schema version doesn't match expected latest, refuse to start (prevent running old schema with new code that assumes migrations ran).

**Modified files:**
- `app_startup.go` — Change `slog.Warn` → return error; add schema version check
- `internal/storage/migrate.go` — Export `LatestVersion()` for startup guard

## Data Flow

```
SettingsPanel/ModelRegistryPanel
  └─ API Key input ─→ Go IPC SaveCredential ─→ CredentialManager
       └─ AES-256-GCM encrypt ─→ SQLite credentials table
       └─ Master key ─→ macOS Keychain / Linux secret-tool

LLM Provider (chat/completion)
  └─ Go IPC GetCredential ─→ CredentialManager
       └─ Decrypt ─→ in-memory key ─→ Python gRPC / direct HTTP
```

## Acceptance Criteria

- [ ] All `llm*Key` fields removed from `settings.ts`; existing localStorage keys are migrated on first read
- [ ] `ModelRegistryPanel` "test connection" works via `GetCredential` (not settings store)
- [ ] `config.yaml` `api_keys` section is zeroed on first startup after migration
- [ ] `Config.GetAPIKey("fred")` returns the value from CredentialManager
- [ ] DB migration failure causes startup to abort with clear error
- [ ] Schema version mismatch causes startup to abort with clear error
- [ ] All existing tests pass; add tests for credential migration paths

## Risks / Trade-offs

- **Migration risk**: Users with existing localStorage API keys will lose them if migration crashes mid-way. Mitigation: migrate on first `GetCredential` call (lazy), not at startup.
- **Keychain dependency**: Linux fallback (`secret-tool`) may not be installed. Document `apt install libsecret-tools` in setup.
- **Offline first**: CredentialManager encrypts/decrypts locally — no network dependency.
