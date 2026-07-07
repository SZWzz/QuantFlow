package auth

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// testMasterKey returns a deterministic MasterKey for use in tests.
func testMasterKey() *MasterKey {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	return NewMasterKey(key)
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		type TEXT NOT NULL,
		data TEXT NOT NULL,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCredentialManager_SaveAndList(t *testing.T) {
	db := setupTestDB(t)
	cm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	err = cm.Save("my-key", "api", map[string]string{"api_key": "sk-12345"})
	if err != nil {
		t.Fatal(err)
	}

	creds, err := cm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Name != "my-key" {
		t.Errorf("name = %q, want 'my-key'", creds[0].Name)
	}
	if creds[0].Keys["api_key"] != "sk-12345" {
		t.Errorf("api_key = %q, want 'sk-12345'", creds[0].Keys["api_key"])
	}
}

func TestCredentialManager_EncryptDecryptRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	cm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	keys := map[string]string{"key1": "value1", "key2": "value2"}
	err = cm.Save("roundtrip", "test", keys)
	if err != nil {
		t.Fatal(err)
	}

	creds, err := cm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Keys["key1"] != "value1" || creds[0].Keys["key2"] != "value2" {
		t.Error("round trip failed: keys don't match")
	}
}

func TestCredentialManager_Delete(t *testing.T) {
	db := setupTestDB(t)
	cm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	cm.Save("delete-me", "api", map[string]string{"k": "v"})
	creds, _ := cm.List()
	if len(creds) != 1 {
		t.Fatal("expected 1 before delete")
	}

	err = cm.Delete("delete-me")
	if err != nil {
		t.Fatal(err)
	}

	creds, err = cm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials after delete, got %d", len(creds))
	}
}

func TestCredentialManager_Names(t *testing.T) {
	db := setupTestDB(t)
	cm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	cm.Save("alpha", "a", map[string]string{"k": "v"})
	cm.Save("beta", "b", map[string]string{"k": "v"})

	names, err := cm.Names()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	if !found["alpha"] || !found["beta"] {
		t.Errorf("names missing: got %v", names)
	}
}

func TestCredentialManager_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	cm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	cm.Save("updatable", "api", map[string]string{"k": "old"})
	cm.Save("updatable", "api", map[string]string{"k": "new"})

	creds, err := cm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Keys["k"] != "new" {
		t.Errorf("expected updated key 'new', got %q", creds[0].Keys["k"])
	}
}

func TestMasterKey_EncryptDecrypt(t *testing.T) {
	key := testMasterKey()
	plaintext := []byte("hello, world!")
	ciphertext, err := key.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := key.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestMasterKey_DifferentKeys(t *testing.T) {
	var key1Arr, key2Arr [32]byte
	for i := range key1Arr {
		key1Arr[i] = 0xAA
		key2Arr[i] = 0xBB
	}
	key1 := NewMasterKey(key1Arr)
	key2 := NewMasterKey(key2Arr)

	plaintext := []byte("secret data")
	ct, err := key1.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	_, err = key2.Decrypt(ct)
	if err == nil {
		t.Error("expected decryption with different key to fail")
	}
}

func TestCredentialManager_OldKeyFallback(t *testing.T) {
	db := setupTestDB(t)

	oldCm, err := newCredentialManager(testMasterKey(), db)
	if err != nil {
		t.Fatal(err)
	}

	err = oldCm.Save("legacy", "api", map[string]string{"token": "legacy-token"})
	if err != nil {
		t.Fatal(err)
	}

	creds, err := oldCm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].Keys["token"] != "legacy-token" {
		t.Errorf("token = %q, want 'legacy-token'", creds[0].Keys["token"])
	}
}
