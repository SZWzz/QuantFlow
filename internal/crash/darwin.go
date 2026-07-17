// internal/crash/darwin.go
//go:build darwin

package crash

import (
	"os"
	"path/filepath"
)

func init() {
	crashLogDir = filepath.Join(os.Getenv("HOME"), "Library", "Logs", "QuantFlow", "crashes")
}
