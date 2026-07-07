# Fix Credential Encryption

## Motivation

`internal/auth/credential.go:60-66` derives the encryption key from `sha256(hostname + salt)`:
- Hostname change (reimage, container) makes all credentials permanently undecryptable
- Any process on the same machine can read the DB and decrypt
- `os.Hostname()` error silently produces a zeroed key

## Design

### Approach: OS keychain (macOS) + file-based fallback

Use the OS-level credential store where available, falling back to encrypted file:

1. **macOS**: Use `security` CLI to store/retrieve a master key in the system Keychain
2. **Linux**: Use `secret-tool` (libsecret) or a key file at `~/.config/quantflow/master.key`
3. **Windows**: Use `CredWrite`/`CredRead` (via syscall) — future work

### Data flow

```
Encrypt:
plaintext → AES-256-GCM(masterKey) → base64 → SQLite

Decrypt:
SQLite → base64 → AES-256-GCM-Decrypt(masterKey) → plaintext

Master key retrieval:
1. Try OS keychain (macOS: security find-generic-password)
2. Fallback: read ~/.config/quantflow/master.key
3. If neither exists: generate random key, store via OS keychain
```

### Modified files

| File | Change |
|------|--------|
| `internal/auth/credential.go` | Replace `deriveKey()` with keychain-backed key management |
| `internal/auth/credential_test.go` | Update tests |
| `internal/auth/master_key.go` | **New** — OS keychain abstraction |
| `internal/auth/master_key_darwin.go` | **New** — macOS Keychain via `security` CLI |
| `internal/auth/master_key_linux.go` | **New** — libsecret / file fallback |
| `go.mod` | No new dependencies (uses `os/exec` to shell out) |

### New types

```go
type MasterKey struct {
    key [32]byte
}

func LoadOrCreateMasterKey() (*MasterKey, error)
func (k *MasterKey) Encrypt(plaintext []byte) ([]byte, error)
func (k *MasterKey) Decrypt(ciphertext []byte) ([]byte, error)
```

`credential.go` uses `MasterKey` instead of `deriveKey()`.

### API changes

- `deriveKey()` removed (unexported, no external impact)
- No changes to `EncryptCredential`/`DecryptCredential` signatures
- No gRPC or frontend changes

## Acceptance Criteria

- [ ] Credential encryption uses OS keychain (macOS: `security add-generic-password`)
- [ ] If keychain unavailable, falls back to `~/.config/quantflow/master.key`
- [ ] Hostname change does not break existing credentials
- [ ] Existing credentials are transparently re-encrypted on next access
- [ ] All credential tests pass
- [ ] `go test -race ./internal/auth/...` passes

## Risks / Trade-offs

- **Keychain CLI dependency**: `security` on macOS, `secret-tool` on Linux. These are standard tools but not guaranteed on minimal containers.
- **Backward compatibility**: Existing credentials encrypted with old scheme must still be decryptable. Migration: try old `deriveKey()` first, then re-encrypt with new key on successful access.
- **User experience**: First launch generates a key silently. No user interaction needed.
