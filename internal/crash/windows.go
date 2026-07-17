// internal/crash/windows.go
//go:build windows

package crash

import (
	"os"
	"path/filepath"
)

func init() {
	crashLogDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "QuantFlow", "crashes")
}
