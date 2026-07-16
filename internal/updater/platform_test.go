package updater

import (
	"testing"
)

func TestPlatformDetection(t *testing.T) {
	// These should not panic — they return the actual OS/arch
	goos := goos()
	goarch := goarch()
	if goos == "unknown" || goarch == "unknown" {
		t.Errorf("platform detection failed: goos=%s goarch=%s", goos, goarch)
	}
}
