package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
)

const defaultCacheSize = 256

// NodeCache provides an LRU cache for node execution results.
// It avoids re-executing expensive nodes when the same inputs are seen again.
type NodeCache struct {
	cache *lru.Cache[string, map[string]any]
}

// NewNodeCache creates a new NodeCache with the given maximum size.
// If size <= 0, defaultCacheSize (256) is used.
func NewNodeCache(size int) (*NodeCache, error) {
	if size <= 0 {
		size = defaultCacheSize
	}
	c, err := lru.New[string, map[string]any](size)
	if err != nil {
		return nil, fmt.Errorf("create lru cache: %w", err)
	}
	return &NodeCache{cache: c}, nil
}

// CacheKey produces a deterministic cache key from a node ID and its inputs
// via JSON serialization (which sorts map keys) followed by SHA-256 hashing.
func CacheKey(nodeID string, inputs map[string]any) string {
	b, err := json.Marshal(inputs)
	if err != nil {
		// Fallback to fmt for un-serializable types.
		b = []byte(fmt.Sprintf("%+v", inputs))
	}
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%s:%x", nodeID, hash[:16])
}

// Get retrieves cached outputs for the given key. The boolean return
// value indicates whether the key was present.
func (c *NodeCache) Get(key string) (map[string]any, bool) {
	return c.cache.Get(key)
}

// Put stores outputs under the given key, evicting the least-recently-used
// entry if the cache is at capacity.
func (c *NodeCache) Put(key string, outputs map[string]any) {
	c.cache.Add(key, outputs)
}

// Len returns the current number of entries in the cache.
func (c *NodeCache) Len() int {
	return c.cache.Len()
}
