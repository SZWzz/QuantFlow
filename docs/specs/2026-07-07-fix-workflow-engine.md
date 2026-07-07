# Fix Workflow Engine Robustness

## Motivation

Four issues in the workflow engine:

1. **NodeContext uses `interface{}`** (`internal/workflow/context.go:9-56`) — 17 service fields are untyped. Any type assertion error causes runtime panic. No compiler validation.

2. **`generateShortID` ignores `rand.Read` error** (`internal/workflow/engine.go:14-18`) — On constrained entropy, produces zeroed ID → all run IDs collide.

3. **Error retry strategy incomplete** (`internal/workflow/engine.go:170,287-291`) — `retryCount` assigned but ignored at layer level; retry only works per-node but not per-layer.

4. **`config.yaml` hardcoded relative path** (`internal/config/config.go:35`) — CWD varies between dev/prod, leading to wrong file location.

## Design

### 1. Type-safe NodeContext via service interfaces

**File**: `internal/workflow/services.go` (new file)

Define typed service interfaces:

```go
package workflow

type MarketHubService interface {
    GetQuote(symbol string) (Quote, error)
    GetOHLCV(symbol string, period string) ([]OHLCV, error)
    // ...
}

type TradingService interface {
    PlaceOrder(order Order) (string, error)
    // ...
}

// One interface per service type — 17 total
```

**File**: `internal/workflow/context.go`

Replace `interface{}` fields with typed interfaces:

```go
type NodeContext struct {
    MarketHub     MarketHubService
    Trading       TradingService
    Portfolio     PortfolioService
    Storage       StorageService
    AIAgent       AIAgentService
    Research      ResearchService
    Notify        NotifyService
    Schedule      ScheduleService
    Logger        LoggerService
    Config        ConfigService
    Broker        BrokerService
    Risk          RiskService
    ML            MLService
    Python        PythonService
    Auth          AuthService
    Cache         CacheService
    Workflow      WorkflowEngineService
}
```

**File**: `app.go` — Update `ServiceStartup` to assign concrete types that implement these interfaces. Each service (e.g., `MarketDataHub`) already has the methods — just declare it implements the interface.

**All node files** (`internal/workflow/nodes/*.go`) — Change type assertions from `.(*market.DataHub)` to `.(workflow.MarketHubService)` or simply use the typed field directly (`ctx.MarketHub.GetQuote(...)`).

### 2. Fix generateShortID

**File**: `internal/workflow/engine.go`

```go
func generateShortID() string {
    b := make([]byte, 4)
    if _, err := rand.Read(b); err != nil {
        // Fallback to timestamp + nanoid
        return fmt.Sprintf("%x", time.Now().UnixNano())
    }
    return hex.EncodeToString(b)
}
```

Also add a uniqueness check — if the generated ID already exists in the workflow's node set, regenerate.

### 3. Fix error retry at layer level

**File**: `internal/workflow/engine.go`

Remove the dead `_ = retryCount` assignment (line 170). At the layer level, implement retry:

```go
func (e *Engine) executeLayer(ctx context.Context, layer *Layer, nodes map[string]*Node) error {
    retryConfig := getNodeErrorConfig(layer.Nodes[0]) // all nodes in layer share config
    maxRetries := 0
    if retryConfig.Strategy == ErrorRetry {
        maxRetries = retryConfig.RetryCount
    }
    // ... execute with retry loop
}
```

### 4. Fix config path resolution

**File**: `internal/config/config.go`

Store the resolved path in the `Config` struct:

```go
type Config struct {
    path string  // resolved absolute path
    // ... existing fields
}

func Load(path string) (*Config, error) {
    // path is resolved by caller (app.go ServiceStartup)
    // ...
}

func (c *Config) Save() error {
    return os.WriteFile(c.path, data, 0644)
}
```

**File**: `app.go` — Resolve config path at startup using `application.AppDir()`:

```go
configPath := filepath.Join(appDir, "config.yaml")
cfg, err := config.Load(configPath)
```

### Modified files

| File | Change |
|------|--------|
| `internal/workflow/services.go` | **New** — typed service interfaces |
| `internal/workflow/context.go` | Replace `interface{}` with typed interfaces |
| `internal/workflow/engine.go` | Fix `generateShortID`, fix error retry |
| `internal/workflow/queue.go` | (updated by shutdown spec) |
| `internal/config/config.go` | Store resolved path, use it in Save |
| `internal/config/config_test.go` | Update tests |
| `app.go` | Resolve config path at startup |
| `internal/workflow/nodes/*.go` | Update all ~85 node files to use typed context fields |

### API changes

- `NodeContext` fields change type from `interface{}` to typed interfaces — **breaking change** for all node implementations
- `config.Load()` now requires a path parameter
- No gRPC or frontend changes

## Acceptance Criteria

- [ ] All 85+ node files compile with typed `NodeContext`
- [ ] No `interface{}` type assertions remain in node `Execute()` methods
- [ ] `generateShortID` handles `rand.Read` error gracefully
- [ ] Layer-level retry actually retries on failure
- [ ] Config saves to correct path in both dev and prod
- [ ] All Go tests pass
- [ ] `go vet ./...` passes with zero issues

## Risks / Trade-offs

- **Breaking change**: All 85+ node files need updates. Each change is mechanical (replace `ctx.Market.(*market.DataHub)` with `ctx.MarketHub`), but the volume is high.
- **Import cycle risk**: Extracting interfaces to `internal/workflow/services.go` may introduce import cycles if service packages import `internal/workflow`. Mitigation: interfaces use only basic types or types from `internal/workflow/types.go`.
- **Config path change**: Backward compatible — if `config.yaml` exists at old relative path, copy it to resolved path on first load.
