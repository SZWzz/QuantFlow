package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

const defaultCacheSize = 256

// CacheKey is a content-addressed hash that recursively includes ancestor inputs.
type CacheKey string

func (k CacheKey) String() string { return string(k) }

// NodeCache provides an LRU cache for node execution results.
// Cache keys are computed recursively: each node's key includes the hash
// of its node type, params, and the cache keys of all upstream connected nodes.
// This enables partial re-execution — only nodes whose transitive inputs changed
// are re-executed.
type NodeCache struct {
	mu       sync.RWMutex
	inner    *lru.Cache[string, map[string]any]
	nodeKeys map[string]CacheKey // nodeID → latest cache key
}

// NewNodeCache creates a new NodeCache with the given maximum size.
func NewNodeCache(size int) (*NodeCache, error) {
	if size <= 0 {
		size = defaultCacheSize
	}
	c, err := lru.New[string, map[string]any](size)
	if err != nil {
		return nil, fmt.Errorf("create lru cache: %w", err)
	}
	return &NodeCache{inner: c, nodeKeys: make(map[string]CacheKey)}, nil
}

// ComputeKey produces a recursive cache key that includes:
//   - node type (class)
//   - all parameter values (sorted by key)
//   - cache keys of all upstream nodes (via edges)
//
// The ancestors map should contain CacheKey values for upstream node IDs.
// This is computed once per engine run and stored in nodeKeys.
func ComputeKey(nodeType string, params map[string]any, ancestors map[string]CacheKey) CacheKey {
	h := sha256.New()
	h.Write([]byte(nodeType))

	// hash params sorted by key for determinism
	pkeys := make([]string, 0, len(params))
	for k := range params {
		pkeys = append(pkeys, k)
	}
	sort.Strings(pkeys)
	for _, k := range pkeys {
		h.Write([]byte(k))
		b, err := json.Marshal(params[k])
		if err != nil {
			b = []byte(fmt.Sprintf("%v", params[k]))
		}
		h.Write(b)
	}

	// recursively include ancestor keys
	akeys := make([]string, 0, len(ancestors))
	for nid, ck := range ancestors {
		akeys = append(akeys, nid+":"+string(ck))
	}
	sort.Strings(akeys)
	for _, s := range akeys {
		h.Write([]byte(s))
	}

	return CacheKey(hex.EncodeToString(h.Sum(nil)))
}

// GetOrCompute returns cached outputs if the node's recursive key matches.
// If cached and key unchanged: returns (outputs, true).
// Otherwise: returns (nil, false).
func (c *NodeCache) GetOrCompute(nodeID, nodeType string, params map[string]any, ancestors map[string]CacheKey) (map[string]any, bool) {
	key := ComputeKey(nodeType, params, ancestors)
	c.mu.RLock()
	existing, ok := c.nodeKeys[nodeID]
	c.mu.RUnlock()
	if ok && existing == key {
		if out, ok := c.inner.Get(string(key)); ok {
			return out, true
		}
	}
	return nil, false
}

// Store caches node outputs and records the cache key for this node.
func (c *NodeCache) Store(nodeID string, key CacheKey, outputs map[string]any) {
	c.mu.Lock()
	c.nodeKeys[nodeID] = key
	c.mu.Unlock()
	c.inner.Add(string(key), outputs)
}

// GetNodeKey returns the recorded cache key for a node, or empty string.
func (c *NodeCache) GetNodeKey(nodeID string) CacheKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nodeKeys[nodeID]
}

// Len returns the current number of entries in the cache.
func (c *NodeCache) Len() int {
	return c.inner.Len()
}
