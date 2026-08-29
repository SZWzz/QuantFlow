package market

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// ErrCacheMiss is returned by Get when the requested key is not in the cache.
var ErrCacheMiss = errors.New("off-hours cache miss")

// OffHoursCache provides sync.Map + atomic JSON persistence for off-hours data.
// Type parameter T must be JSON-serializable.
type OffHoursCache[T any] struct {
	mu   sync.Mutex
	data map[string]T
	path string
	name string
}

func NewOffHoursCache[T any](name string) *OffHoursCache[T] {
	return &OffHoursCache[T]{
		data: make(map[string]T),
		name: name,
	}
}

func (c *OffHoursCache[T]) SetPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.path = path
}

func (c *OffHoursCache[T]) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.path == "" {
		return nil
	}
	b, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var data map[string]T
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	c.data = data
	slog.Info("loaded off-hours cache", "name", c.name, "count", len(data), "path", c.path)
	return nil
}

func (c *OffHoursCache[T]) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.path == "" {
		return nil
	}
	if len(c.data) == 0 {
		return nil
	}
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, c.path); err != nil {
		return err
	}
	slog.Debug("saved off-hours cache", "name", c.name, "count", len(c.data))
	return nil
}

func (c *OffHoursCache[T]) Get(key string, dest *T) error {
	c.mu.Lock()
	v, ok := c.data[key]
	c.mu.Unlock()
	if !ok {
		return ErrCacheMiss
	}
	// Deep-copy via JSON to prevent caller mutation of cached data
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *OffHoursCache[T]) Set(key string, val T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = val
}

func (c *OffHoursCache[T]) GetAll() map[string]T {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make(map[string]T, len(c.data))
	for k, v := range c.data {
		cp[k] = v
	}
	return cp
}

func (c *OffHoursCache[T]) SetAll(data map[string]T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
}
