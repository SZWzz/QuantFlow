# Implementation Plan: Recursive Cache Key Signature for Partial Re-execution

## Step 1: Rewrite cache key computation

**File**: `internal/workflow/cache.go`

Replace the entire file with:

```go
package workflow

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "sort"
)

// CacheKey is a content-addressed hash that recursively includes ancestor inputs.
type CacheKey string

func (c CacheKey) String() string { return string(c) }

// CacheEntry stores a node's cached output.
type CacheEntry struct {
    Key     CacheKey
    Outputs map[string]any
}

// CacheStore manages cache entries with recursive key support.
type CacheStore struct {
    entries  map[CacheKey]*CacheEntry
    nodeKeys map[string]CacheKey // nodeID → latest cache key
}

func NewCacheStore() *CacheStore {
    return &CacheStore{
        entries:  make(map[CacheKey]*CacheEntry),
        nodeKeys: make(map[string]CacheKey),
    }
}

// ComputeKey computes a recursive cache key for a node.
// It includes the node's type, parameters, and the keys of all ancestor nodes
// connected via edges, enabling partial re-execution detection.
func ComputeKey(nodeType string, params map[string]any, edges []Edge, nodeID string, nodeResults map[string]*CacheEntry) CacheKey {
    h := sha256.New()
    h.Write([]byte(nodeType))

    // params sorted by key for determinism
    pkeys := make([]string, 0, len(params))
    for k := range params {
        pkeys = append(pkeys, k)
    }
    sort.Strings(pkeys)
    for _, k := range pkeys {
        h.Write([]byte(k))
        b, _ := json.Marshal(params[k])
        h.Write(b)
    }

    // Connected inputs: recursively include source node's cache key
    for _, edge := range edges {
        if edge.ToNode == nodeID {
            if srcEntry, ok := nodeResults[edge.FromNode]; ok {
                h.Write([]byte("link:"))
                h.Write([]byte(srcEntry.Key))
            }
        }
    }

    return CacheKey(hex.EncodeToString(h.Sum(nil)))
}

// GetOrCompute returns cached output if key matches, or nil for cache miss.
func (s *CacheStore) GetOrCompute(nodeID, nodeType string, params map[string]any, edges []Edge, allNodes map[string]BaseNode) (map[string]any, bool) {
    key := ComputeKey(nodeType, params, edges, nodeID, allNodes, s.entries)
    if existing, ok := s.nodeKeys[nodeID]; ok && existing == key {
        if entry, ok := s.entries[key]; ok {
            return entry.Outputs, true
        }
    }
    return nil, false
}

func (s *CacheStore) Store(nodeID string, key CacheKey, outputs map[string]any) {
    s.entries[key] = &CacheEntry{Key: key, Outputs: outputs}
    s.nodeKeys[nodeID] = key
}
```

Commit: `feat(workflow): replace flat cache key with recursive input signature`
