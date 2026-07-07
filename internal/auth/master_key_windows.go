//go:build windows

package auth

import "fmt"

// LoadOrCreateMasterKey is not yet supported on Windows.
func LoadOrCreateMasterKey() (*MasterKey, error) {
	return nil, fmt.Errorf("Windows keychain not yet supported")
}
