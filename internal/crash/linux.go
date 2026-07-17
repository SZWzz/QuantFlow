// internal/crash/linux.go
//go:build linux

package crash

import (
	"os"
	"path/filepath"
)

func init() {
	crashLogDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "QuantFlow", "crashes")
}
