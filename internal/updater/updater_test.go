// internal/updater/updater_test.go
package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionComparison(t *testing.T) {
	u := &Updater{}
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"2026.7.14", "2026.7.20", true},
		{"2026.7.20", "2026.7.20", false},
		{"2026.8.1", "2026.7.20", false},
		{"2025.12.31", "2026.1.1", true},
	}
	for _, tt := range tests {
		got := u.isNewer(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestChecksumVerify(t *testing.T) {
	content := []byte("test binary content")
	sum := sha256.Sum256(content)
	expected := hex.EncodeToString(sum[:])

	tmp := filepath.Join(t.TempDir(), "test.bin")
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		t.Fatal(err)
	}

	u := &Updater{}
	if err := u.Verify(tmp, expected); err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	if err := u.Verify(tmp, "badchecksum"); err == nil {
		t.Error("expected error for bad checksum")
	}
}
