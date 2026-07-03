# Recursive Cache Key Signature for Partial Re-execution

## Motivation
Current cache uses `SHA256(nodeID + json(inputs))` which is a flat hash. If any ancestor node changes, the cached entry for downstream nodes becomes stale but the current LRU cache doesn't detect this. ComfyUI's `CacheKeySetInputSignature` recursively hashes the transitive input ancestry, enabling partial re-execution: only nodes whose inputs actually changed are re-executed.

## Design

Replace current `CacheKey` function with a recursive hash that includes:
1. Node's class type
2. Node's parameter values
3. For connected inputs: the source node's cache key (recursively)
4. For literal inputs: the literal value hash

```go
type CacheKey string

func ComputeCacheKey(node BaseNode, params map[string]any, inputs map[string]InputSource, cache *CacheStore) CacheKey {
    h := sha256.New()
    h.Write([]byte(node.NodeType()))
    // hash params sorted by key
    keys := sortedKeys(params)
    for _, k := range keys {
        h.Write([]byte(k))
        h.Write([]byte(fmt.Sprintf("%v", params[k])))
    }
    // hash input sources recursively
    for _, name := range sortedKeys(inputs) {
        src := inputs[name]
        if src.IsLink {
            childKey := cache.GetKey(src.NodeID)
            h.Write([]byte(childKey))
        } else {
            h.Write([]byte(fmt.Sprintf("%v", src.LiteralValue)))
        }
    }
    return CacheKey(hex.EncodeToString(h.Sum(nil)))
}
```

**Changed files:**
- `internal/workflow/cache.go` — replace cache key computation
- `internal/workflow/cache.go` — add recursive key storage

## Acceptance Criteria
- [ ] Changing an upstream node's params invalidates downstream cache
- [ ] Changing an unrelated branch does NOT invalidate cache
- [ ] New cache key is deterministic (same inputs = same key)
- [ ] All existing tests pass

## Risks / Trade-offs
Slightly more expensive key computation (recursive traversal). Mitigated by memoisation of per-node keys in `nodeKeys` map.
