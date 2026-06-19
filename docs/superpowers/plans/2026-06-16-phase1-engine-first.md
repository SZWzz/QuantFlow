# Phase 1: Engine-First — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a pure-Go workflow engine with CLI — DAG validation, parallel goroutine execution, SQLite persistence, and 80%+ test coverage.

**Architecture:** Five serial milestones. M1 scaffolds the Go project with config/logging/SQLite. M2 defines the node interface, registry, and 5 reference node implementations. M3 builds the DAG execution engine (topo sort + goroutine layers + caching). M4 adds persistence with versioned storage and breakpoint recovery. M5 rounds out the CLI, test suite, benchmarks, and example workflows.

**Tech Stack:** Go 1.22+, SQLite (mattn/go-sqlite3, WAL mode), Viper (config), slog (logging), golang.org/x/sync/errgroup, hashicorp/golang-lru (cache)

**Spec:** [docs/superpowers/specs/2026-06-16-phase1-engine-first-design.md](../specs/2026-06-16-phase1-engine-first-design.md)

---

## File Map

```
app/
├── main.go                              # Entry point → calls cmd/qf
├── go.mod / go.sum
├── Makefile
├── config.yaml                           # Default config
└── internal/
    ├── config/
    │   └── config.go                     # Viper loader
    ├── logging/
    │   └── logging.go                    # slog Setup()
    ├── storage/
    │   ├── db.go                         # Open() — WAL mode SQLite
    │   ├── migrate.go                    # Run(version) migration runner
    │   ├── workflow_repo.go              # WorkflowRepo CRUD
    │   └── migrations/
    │       ├── 001_init.sql              # placeholder table
    │       ├── 002_workflows.sql         # workflows + versions + exec_history
    │       └── 003_checkpoints.sql       # execution_checkpoints
    └── workflow/
        ├── node.go                       # PortType, PortDefinition, ParamDef, BaseNode
        ├── registry.go                   # NodeRegistry
        ├── registry_test.go
        ├── dag.go                        # Edge, NodeInstance, Workflow, TopoSort, Validate
        ├── dag_test.go
        ├── engine.go                     # Engine, ExecutionResult, Execute()
        ├── engine_test.go
        ├── cache.go                      # LRU wrapper
        └── nodes/
            ├── data_loader.go            # DataLoaderNode
            ├── data_loader_test.go
            ├── sma.go                    # SMANode
            ├── sma_test.go
            ├── cross_signal.go           # CrossSignalNode
            ├── cross_signal_test.go
            ├── log_output.go             # LogOutputNode
            ├── log_output_test.go
            ├── loop.go                   # LoopNode
            └── loop_test.go

cmd/
└── qf/
    └── main.go                           # CLI (run, nodes, validate, version)

examples/
├── sma_cross.json                        # SMA cross-over workflow
├── multi_asset.json                      # Loop over multiple symbols
└── error_handling.json                   # Node failure demo
```

---

## Milestone 1: Go 项目地基

### Task 1: Initialize Go module and directory structure

**Files:**
- Create: `app/go.mod`
- Create: `app/main.go`

- [ ] **Step 1: Create go.mod**

```bash
cd app && go mod init quantflow
```

- [ ] **Step 2: Create minimal main.go**

Create `app/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"quantflow/internal/config"
	"quantflow/internal/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	logging.Setup(cfg.LogLevel)
	fmt.Printf("quantflow %s starting...\n", cfg.Version)
}
```

- [ ] **Step 3: Create directory structure**

```bash
mkdir -p app/internal/config app/internal/logging app/internal/storage/migrations app/internal/workflow/nodes
mkdir -p cmd/qf
mkdir -p examples
```

- [ ] **Step 4: Verify it fails to build (missing packages)**

Run: `cd app && go build ./...`
Expected: errors about missing config and logging packages

- [ ] **Step 5: Commit**

```bash
git add app/go.mod app/main.go
git commit -m "feat(m1): init go module and directory skeleton"
```

---

### Task 2: Config package

**Files:**
- Create: `app/internal/config/config.go`
- Create: `app/config.yaml`

- [ ] **Step 1: Write the test**

Create `app/internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.yaml")
	os.WriteFile(cfgPath, []byte(`version: "0.0.1"
log_level: "info"
db_path: "data/quantflow.db"`), 0644)

	cfg, err := loadFile(cfgPath)
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}
	if cfg.Version != "0.0.1" {
		t.Errorf("version = %q, want %q", cfg.Version, "0.0.1")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("log_level = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.DBPath != "data/quantflow.db" {
		t.Errorf("db_path = %q, want %q", cfg.DBPath, "data/quantflow.db")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := loadFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/config/ -v`
Expected: FAIL — `loadFile` not defined

- [ ] **Step 3: Implement config package**

Create `app/internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
	DBPath   string `yaml:"db_path"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Version:  "0.0.1",
		LogLevel: "info",
		DBPath:   "data/quantflow.db",
	}
}

// Load reads config from default locations, falling back to defaults.
func Load() (*Config, error) {
	paths := []string{"config.yaml", "config/config.yaml"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return loadFile(p)
		}
	}
	return DefaultConfig(), nil
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 5: Install yaml dependency**

```bash
cd app && go get gopkg.in/yaml.v3
```

- [ ] **Step 6: Create default config file**

Create `app/config.yaml`:

```yaml
version: "0.0.1"
log_level: "info"
db_path: "data/quantflow.db"
```

- [ ] **Step 7: Commit**

```bash
git add app/internal/config/ app/config.yaml app/go.mod app/go.sum
git commit -m "feat(m1): add config package with viper/yaml loader"
```

---

### Task 3: Logging package

**Files:**
- Create: `app/internal/logging/logging.go`

- [ ] **Step 1: Implement logging package**

Create `app/internal/logging/logging.go`:

```go
// Package logging provides a thin wrapper around slog for application-level logging.
package logging

import (
	"log/slog"
	"os"
)

// Setup configures the default slog logger with the given level.
// Valid levels: "debug", "info", "warn", "error".
func Setup(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
```

- [ ] **Step 2: Verify build**

Run: `cd app && go build ./...`
Expected: builds successfully

- [ ] **Step 3: Commit**

```bash
git add app/internal/logging/logging.go
git commit -m "feat(m1): add logging package (slog wrapper)"
```

---

### Task 4: Storage — SQLite connection

**Files:**
- Create: `app/internal/storage/db.go`
- Create: `app/internal/storage/db_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/storage/db_test.go`:

```go
package storage

import (
	"testing"
)

func TestOpen_Pragmas(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Verify WAL mode
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode error = %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	// Verify foreign keys
	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys error = %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("foreign_keys = %d, want 1", fkEnabled)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/storage/ -v`
Expected: FAIL — `Open` not defined

- [ ] **Step 3: Implement Open**

Create `app/internal/storage/db.go`:

```go
// Package storage manages the SQLite database connection and migrations.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Open creates/opens the SQLite database at the given path with WAL mode
// and foreign keys enabled.
func Open(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}
```

- [ ] **Step 4: Install sqlite3 driver**

```bash
cd app && go get github.com/mattn/go-sqlite3
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd app && go test ./internal/storage/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/internal/storage/db.go app/internal/storage/db_test.go app/go.mod app/go.sum
git commit -m "feat(m1): add sqlite connection with WAL mode"
```

---

### Task 5: Migration framework

**Files:**
- Create: `app/internal/storage/migrate.go`
- Create: `app/internal/storage/migrate_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/storage/migrate_test.go`:

```go
package storage

import (
	"testing"
)

func TestRunMigrations_CreatesVersionTable(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	migrations := []Migration{
		{Version: 1, SQL: "CREATE TABLE test_t (id INTEGER PRIMARY KEY)"},
	}

	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify the table was created
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM test_t").Scan(&count); err != nil {
		t.Errorf("test_t should exist: %v", err)
	}

	// Run again — should be idempotent
	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() second call error = %v", err)
	}
}

func TestRunMigrations_AppliesInOrder(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	migrations := []Migration{
		{Version: 1, SQL: "CREATE TABLE t1 (id INTEGER PRIMARY KEY)"},
		{Version: 2, SQL: "CREATE TABLE t2 (id INTEGER PRIMARY KEY)"},
		{Version: 3, SQL: "INSERT INTO t2 VALUES (1)"},
	}

	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Both tables should exist
	for _, table := range []string{"t1", "t2"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Errorf("%s should exist: %v", table, err)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/storage/ -v -run TestRun`
Expected: FAIL — `Run` and `Migration` not defined

- [ ] **Step 3: Implement migration framework**

Create `app/internal/storage/migrate.go`:

```go
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migration represents a numbered schema migration.
type Migration struct {
	Version int
	SQL     string
}

// Run applies all pending migrations in version order.
func Run(db *sql.DB, migrations []Migration) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	for _, m := range migrations {
		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_version WHERE version = ?", m.Version).Scan(&exists); err != nil {
			return fmt.Errorf("check version %d: %w", m.Version, err)
		}
		if exists > 0 {
			continue
		}

		if _, err := db.Exec(m.SQL); err != nil {
			return fmt.Errorf("migration %d: %w", m.Version, err)
		}

		if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (?)", m.Version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}

		slog.Info("migration applied", "version", m.Version)
	}

	return nil
}

// BuiltinMigrations returns all embedded migrations sorted by version.
func BuiltinMigrations() ([]Migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var migrations []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("parse version from %s: %w", e.Name(), err)
		}
		sql, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		migrations = append(migrations, Migration{Version: version, SQL: string(sql)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/storage/ -v -run TestRun`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/storage/migrate.go app/internal/storage/migrate_test.go
git commit -m "feat(m1): add migration framework with version tracking"
```

---

### Task 6: Migration 001 — init

**Files:**
- Create: `app/internal/storage/migrations/001_init.sql`

- [ ] **Step 1: Create init migration**

Create `app/internal/storage/migrations/001_init.sql`:

```sql
-- 001_init: placeholder initial schema.
-- The schema_version table is created by the migration framework itself.
-- This migration exists to validate the migration pipeline.
SELECT 1;
```

- [ ] **Step 2: Update main.go to run migrations on startup**

Edit `app/main.go`:

```go
package main

import (
	"fmt"
	"log/slog"
	"os"

	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	logging.Setup(cfg.LogLevel)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrations, err := storage.BuiltinMigrations()
	if err != nil {
		slog.Error("failed to load migrations", "error", err)
		os.Exit(1)
	}
	if err := storage.Run(db, migrations); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	fmt.Printf("quantflow %s started\n", cfg.Version)
}
```

- [ ] **Step 3: Verify build and test**

```bash
cd app && go build ./... && go test ./internal/storage/ -v
```

- [ ] **Step 4: Commit**

```bash
git add app/internal/storage/migrations/001_init.sql app/main.go
git commit -m "feat(m1): add embedded migration 001_init and wire startup"
```

---

### Task 7: Makefile

**Files:**
- Create: `app/Makefile`

- [ ] **Step 1: Create Makefile**

Create `app/Makefile`:

```makefile
.PHONY: build test lint vet clean bench coverage

build:
	go build ./...

test:
	go test ./... -v -count=1

test-race:
	go test -race ./... -v -count=1

bench:
	go test ./... -bench=. -benchmem

lint:
	go vet ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out
	rm -rf data/
```

- [ ] **Step 2: Verify make targets**

```bash
cd app && make build && make test && make lint
```
Expected: all pass

- [ ] **Step 3: Commit**

```bash
git add app/Makefile
git commit -m "feat(m1): add Makefile with build/test/lint/bench targets"
```

---

## Milestone 2: 节点系统

### Task 8: Node types — BaseNode, PortType, ParamDef

**Files:**
- Create: `app/internal/workflow/node.go`
- Create: `app/internal/workflow/node_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/workflow/node_test.go`:

```go
package workflow

import (
	"testing"
)

func TestPortTypeConstants(t *testing.T) {
	types := map[PortType]bool{
		PortOHLCV:  true,
		PortSeries: true,
		PortSignal: true,
		PortString: true,
		PortAny:    true,
	}
	for pt, ok := range types {
		if !ok {
			t.Errorf("PortType %q should exist", pt)
		}
	}
}

func TestPortDefinition(t *testing.T) {
	pd := PortDefinition{Name: "close_price", Type: PortSeries, Required: true}
	if pd.Name != "close_price" {
		t.Errorf("Name = %q, want %q", pd.Name, "close_price")
	}
	if pd.Type != PortSeries {
		t.Errorf("Type = %q, want %q", pd.Type, PortSeries)
	}
	if !pd.Required {
		t.Error("Required should be true")
	}
}

func TestParamDef(t *testing.T) {
	pd := ParamDef{Name: "period", Type: "int", Default: 20, Description: "SMA window"}
	if pd.Name != "period" {
		t.Errorf("Name = %q, want %q", pd.Name, "period")
	}
	if pd.Default != 20 {
		t.Errorf("Default = %v, want 20", pd.Default)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/workflow/ -v -run TestPort`
Expected: FAIL — types not defined

- [ ] **Step 3: Implement node types**

Create `app/internal/workflow/node.go`:

```go
// Package workflow provides the workflow engine — DAG execution, node registry,
// and the BaseNode interface that all nodes implement.
package workflow

import "context"

// PortType defines the data type flowing through ports.
type PortType string

const (
	PortOHLCV  PortType = "ohlcv"
	PortSeries PortType = "series" // []float64
	PortSignal PortType = "signal" // buy/sell/hold + confidence
	PortString PortType = "string"
	PortAny    PortType = "any"
)

// PortDefinition describes an input or output port.
type PortDefinition struct {
	Name     string   `json:"name"`
	Type     PortType `json:"type"`
	Required bool     `json:"required"`
}

// ParamDef describes a configurable parameter for a node.
type ParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "int", "float", "string", "bool", "string_array"
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// BaseNode is the interface every workflow node must implement.
type BaseNode interface {
	ID() string
	NodeType() string
	Category() string
	InputPorts() []PortDefinition
	OutputPorts() []PortDefinition
	ParamSchema() []ParamDef
	Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error)
	Validate() error
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/workflow/ -v -run TestPort`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/workflow/node.go app/internal/workflow/node_test.go
git commit -m "feat(m2): define BaseNode interface, PortType, ParamDef"
```

---

### Task 9: NodeRegistry

**Files:**
- Create: `app/internal/workflow/registry.go`
- Create: `app/internal/workflow/registry_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/workflow/registry_test.go`:

```go
package workflow

import (
	"context"
	"testing"
)

// stubNode is a minimal BaseNode for testing the registry.
type stubNode struct {
	id       string
	nodeType string
	cat      string
	params   map[string]any
}

func (s *stubNode) ID() string                       { return s.id }
func (s *stubNode) NodeType() string                  { return s.nodeType }
func (s *stubNode) Category() string                  { return s.cat }
func (s *stubNode) InputPorts() []PortDefinition      { return nil }
func (s *stubNode) OutputPorts() []PortDefinition     { return nil }
func (s *stubNode) ParamSchema() []ParamDef           { return nil }
func (s *stubNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	return nil, nil
}
func (s *stubNode) Validate() error { return nil }

func newStubNode(id string, params map[string]any) (BaseNode, error) {
	return &stubNode{id: id, nodeType: "stub", cat: "test", params: params}, nil
}

func TestRegistry_RegisterAndCreate(t *testing.T) {
	r := NewRegistry()
	r.Register("stub", newStubNode)

	node, err := r.Create("stub", "node1", map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if node.ID() != "node1" {
		t.Errorf("ID() = %q, want %q", node.ID(), "node1")
	}
	if node.NodeType() != "stub" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "stub")
	}
}

func TestRegistry_CreateUnknownType(t *testing.T) {
	r := NewRegistry()
	_, err := r.Create("nonexistent", "n1", nil)
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register("stub_a", newStubNode)
	r.Register("stub_b", newStubNode)

	all := r.ListAll()
	if len(all) != 2 {
		t.Errorf("ListAll() len = %d, want 2", len(all))
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	r := NewRegistry()
	r.Register("dup", newStubNode)
	r.Register("dup", newStubNode)
	node, err := r.Create("dup", "n", nil)
	if err != nil {
		t.Fatalf("Create() after re-register error = %v", err)
	}
	if node == nil {
		t.Error("node should not be nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/workflow/ -v -run TestRegistry`
Expected: FAIL — `NewRegistry` not defined

- [ ] **Step 3: Implement NodeRegistry**

Create `app/internal/workflow/registry.go`:

```go
package workflow

import (
	"fmt"
	"sync"
)

// NodeConstructor is a factory function that creates a node instance.
type NodeConstructor func(id string, params map[string]any) (BaseNode, error)

// NodeMeta holds metadata about a registered node type.
type NodeMeta struct {
	NodeType string `json:"node_type"`
	Category string `json:"category"`
}

// NodeRegistry manages node type registration and instantiation.
// It is safe for concurrent use.
type NodeRegistry struct {
	mu           sync.RWMutex
	constructors map[string]NodeConstructor
	categories   map[string]string
}

// NewRegistry creates an empty NodeRegistry.
func NewRegistry() *NodeRegistry {
	return &NodeRegistry{
		constructors: make(map[string]NodeConstructor),
		categories:   make(map[string]string),
	}
}

// Register adds a node type constructor.
func (r *NodeRegistry) Register(nodeType string, ctor NodeConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[nodeType] = ctor
}

// RegisterWithCategory registers a node type with its category metadata.
func (r *NodeRegistry) RegisterWithCategory(nodeType string, ctor NodeConstructor, category string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[nodeType] = ctor
	r.categories[nodeType] = category
}

// Create instantiates a node of the given type.
func (r *NodeRegistry) Create(nodeType string, id string, params map[string]any) (BaseNode, error) {
	r.mu.RLock()
	ctor, ok := r.constructors[nodeType]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown node type: %q", nodeType)
	}
	return ctor(id, params)
}

// ListAll returns metadata for all registered node types.
func (r *NodeRegistry) ListAll() []NodeMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []NodeMeta
	for nodeType, cat := range r.categories {
		result = append(result, NodeMeta{NodeType: nodeType, Category: cat})
	}
	return result
}

// Has returns true if the node type is registered.
func (r *NodeRegistry) Has(nodeType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.constructors[nodeType]
	return ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/workflow/ -v -run TestRegistry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/workflow/registry.go app/internal/workflow/registry_test.go
git commit -m "feat(m2): add NodeRegistry with thread-safe type registration"
```

---

### Task 10: SMANode — first node implementation

**Files:**
- Create: `app/internal/workflow/nodes/sma.go`
- Create: `app/internal/workflow/nodes/sma_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/workflow/nodes/sma_test.go`:

```go
package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestSMANode_Execute(t *testing.T) {
	node := NewSMANode("sma1", map[string]any{"period": float64(3)})
	if node.ID() != "sma1" {
		t.Errorf("ID() = %q, want %q", node.ID(), "sma1")
	}
	if node.NodeType() != "sma" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "sma")
	}
	if node.Category() != "indicator" {
		t.Errorf("Category() = %q, want %q", node.Category(), "indicator")
	}

	inputs := map[string]any{"input": []float64{1.0, 2.0, 3.0, 4.0, 5.0}}
	params := map[string]any{"period": float64(3)}

	outputs, err := node.Execute(context.Background(), inputs, params)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result, ok := outputs["output"].([]float64)
	if !ok {
		t.Fatalf("output is %T, want []float64", outputs["output"])
	}
	expected := []float64{1, 1.5, 2, 3, 4}
	if len(result) != len(expected) {
		t.Fatalf("len = %d, want %d", len(result), len(expected))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("[%d] = %v, want %v", i, result[i], v)
		}
	}
}

func TestSMANode_PortDefinitions(t *testing.T) {
	node := NewSMANode("sma1", nil)
	inputs := node.InputPorts()
	if len(inputs) != 1 || inputs[0].Name != "input" {
		t.Errorf("InputPorts: %+v, want 1 port named 'input'", inputs)
	}
	if inputs[0].Type != workflow.PortSeries {
		t.Errorf("input port type = %q, want %q", inputs[0].Type, workflow.PortSeries)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/workflow/nodes/ -v -run TestSMA`
Expected: FAIL — `NewSMANode` not defined

- [ ] **Step 3: Implement SMANode**

Create `app/internal/workflow/nodes/sma.go`:

```go
// Package nodes provides built-in workflow node implementations.
package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// SMANode computes a Simple Moving Average over its input series.
type SMANode struct {
	id     string
	params map[string]any
}

// NewSMANode creates a new SMANode.
func NewSMANode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SMANode{id: id, params: params}, nil
}

func (n *SMANode) ID() string       { return n.id }
func (n *SMANode) NodeType() string { return "sma" }
func (n *SMANode) Category() string { return "indicator" }

func (n *SMANode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "input", Type: workflow.PortSeries, Required: true},
	}
}

func (n *SMANode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "output", Type: workflow.PortSeries, Required: false},
	}
}

func (n *SMANode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "period", Type: "int", Default: 20, Description: "SMA window size"},
	}
}

func (n *SMANode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	raw, ok := inputs["input"]
	if !ok {
		return nil, fmt.Errorf("sma: missing required input 'input'")
	}
	series, ok := toFloat64Slice(raw)
	if !ok {
		return nil, fmt.Errorf("sma: input must be []float64, got %T", raw)
	}

	period := 20
	if p, ok := params["period"]; ok {
		switch v := p.(type) {
		case float64:
			period = int(v)
		case int:
			period = v
		}
	}

	result := sma(series, period)
	return map[string]any{"output": result}, nil
}

func (n *SMANode) Validate() error {
	period := 20
	if p, ok := n.params["period"]; ok {
		switch v := p.(type) {
		case float64:
			period = int(v)
		case int:
			period = v
		}
	}
	if period <= 0 {
		return fmt.Errorf("sma: period must be positive, got %d", period)
	}
	return nil
}

// sma computes the simple moving average.
// For the first (period-1) elements, returns the mean of available values.
func sma(data []float64, period int) []float64 {
	if len(data) == 0 || period <= 0 {
		return nil
	}
	result := make([]float64, len(data))
	var sum float64
	for i, v := range data {
		sum += v
		if i < period {
			result[i] = sum / float64(i+1)
		} else {
			sum -= data[i-period]
			result[i] = sum / float64(period)
		}
	}
	return result
}

// toFloat64Slice attempts to convert an any to []float64.
func toFloat64Slice(raw any) ([]float64, bool) {
	switch v := raw.(type) {
	case []float64:
		return v, true
	case []any:
		result := make([]float64, len(v))
		for i, elem := range v {
			f, ok := toFloat64(elem)
			if !ok {
				return nil, false
			}
			result[i] = f
		}
		return result, true
	default:
		return nil, false
	}
}

// toFloat64 attempts to convert an any to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/workflow/nodes/ -v -run TestSMA`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/workflow/nodes/sma.go app/internal/workflow/nodes/sma_test.go
git commit -m "feat(m2): implement SMANode with simple moving average"
```

---

### Task 11: DataLoaderNode

**Files:**
- Create: `app/internal/workflow/nodes/data_loader.go`
- Create: `app/internal/workflow/nodes/data_loader_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/workflow/nodes/data_loader_test.go`:

```go
package nodes

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDataLoaderNode_CSV(t *testing.T) {
	tmp := t.TempDir()
	csvPath := filepath.Join(tmp, "test.csv")
	os.WriteFile(csvPath, []byte("date,open,high,low,close,volume\n2024-01-01,100,110,95,105,1000\n2024-01-02,105,115,100,110,1200\n"), 0644)

	node, err := NewDataLoaderNode("loader1", map[string]any{"source": "csv", "path": csvPath})
	if err != nil {
		t.Fatalf("NewDataLoaderNode() error = %v", err)
	}

	outputs, err := node.Execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	ohlcv, ok := outputs["ohlcv"]
	if !ok {
		t.Fatal("missing 'ohlcv' output")
	}
	data, ok := ohlcv.([]OHLCVBar)
	if !ok {
		t.Fatalf("ohlcv is %T, want []OHLCVBar", ohlcv)
	}
	if len(data) != 2 {
		t.Fatalf("len = %d, want 2", len(data))
	}
	if data[0].Close != 105 {
		t.Errorf("data[0].Close = %v, want 105", data[0].Close)
	}
	if data[1].Volume != 1200 {
		t.Errorf("data[1].Volume = %v, want 1200", data[1].Volume)
	}
}

func TestDataLoaderNode_PortDefinitions(t *testing.T) {
	node, _ := NewDataLoaderNode("dl", map[string]any{"source": "csv", "path": "dummy.csv"})
	inputs := node.InputPorts()
	if len(inputs) != 0 {
		t.Errorf("InputPorts should be empty, got %d", len(inputs))
	}
	outputs := node.OutputPorts()
	if len(outputs) != 1 || outputs[0].Name != "ohlcv" {
		t.Errorf("OutputPorts: %+v, want 1 port named 'ohlcv'", outputs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/workflow/nodes/ -v -run TestDataLoader`
Expected: FAIL

- [ ] **Step 3: Implement DataLoaderNode**

Create `app/internal/workflow/nodes/data_loader.go`:

```go
package nodes

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"quantflow/internal/workflow"
)

// OHLCVBar represents a single OHLCV candle.
type OHLCVBar struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// DataLoaderNode loads OHLCV data from CSV files.
type DataLoaderNode struct {
	id     string
	params map[string]any
}

// NewDataLoaderNode creates a new DataLoaderNode.
func NewDataLoaderNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &DataLoaderNode{id: id, params: params}, nil
}

func (n *DataLoaderNode) ID() string       { return n.id }
func (n *DataLoaderNode) NodeType() string { return "data_loader" }
func (n *DataLoaderNode) Category() string { return "data" }

func (n *DataLoaderNode) InputPorts() []workflow.PortDefinition {
	return nil
}

func (n *DataLoaderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv", Type: workflow.PortOHLCV, Required: false},
	}
}

func (n *DataLoaderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "source", Type: "string", Default: "csv", Description: "Data source type"},
		{Name: "path", Type: "string", Default: "", Description: "Path to CSV file"},
	}
}

func (n *DataLoaderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	source := "csv"
	path := ""
	if s, ok := n.params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if p, ok := n.params["path"]; ok {
		path = fmt.Sprint(p)
	}
	if s, ok := params["source"]; ok {
		source = fmt.Sprint(s)
	}
	if p, ok := params["path"]; ok {
		path = fmt.Sprint(p)
	}

	switch source {
	case "csv":
		bars, err := loadCSV(path)
		if err != nil {
			return nil, fmt.Errorf("data_loader: %w", err)
		}
		return map[string]any{"ohlcv": bars}, nil
	default:
		return nil, fmt.Errorf("data_loader: unknown source %q", source)
	}
}

func (n *DataLoaderNode) Validate() error {
	path := ""
	if p, ok := n.params["path"]; ok {
		path = fmt.Sprint(p)
	}
	if path == "" {
		return fmt.Errorf("data_loader: path is required")
	}
	return nil
}

func loadCSV(path string) ([]OHLCVBar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[col] = i
	}

	var bars []OHLCVBar
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row: %w", err)
		}
		bar := OHLCVBar{
			Date:   record[colIdx["date"]],
			Open:   parseFloat(record[colIdx["open"]]),
			High:   parseFloat(record[colIdx["high"]]),
			Low:    parseFloat(record[colIdx["low"]]),
			Close:  parseFloat(record[colIdx["close"]]),
			Volume: parseFloat(record[colIdx["volume"]]),
		}
		bars = append(bars, bar)
	}
	return bars, nil
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/workflow/nodes/ -v -run TestDataLoader`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/workflow/nodes/data_loader.go app/internal/workflow/nodes/data_loader_test.go
git commit -m "feat(m2): implement DataLoaderNode with CSV support"
```

---

### Task 12: CrossSignalNode, LogOutputNode, LoopNode

**Files:**
- Create: `app/internal/workflow/nodes/cross_signal.go`
- Create: `app/internal/workflow/nodes/cross_signal_test.go`
- Create: `app/internal/workflow/nodes/log_output.go`
- Create: `app/internal/workflow/nodes/log_output_test.go`
- Create: `app/internal/workflow/nodes/loop.go`
- Create: `app/internal/workflow/nodes/loop_test.go`

- [ ] **Step 1: Implement CrossSignalNode with test**

Create `app/internal/workflow/nodes/cross_signal.go`:

```go
package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// CrossSignalNode detects when a fast MA crosses over a slow MA.
type CrossSignalNode struct {
	id     string
	params map[string]any
}

// SignalEvent represents a trading signal.
type SignalEvent struct {
	Index      int     `json:"index"`
	Direction  string  `json:"direction"` // "buy" or "sell"
	FastValue  float64 `json:"fast_value"`
	SlowValue  float64 `json:"slow_value"`
	Confidence float64 `json:"confidence"`
}

// NewCrossSignalNode creates a new CrossSignalNode.
func NewCrossSignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &CrossSignalNode{id: id, params: params}, nil
}

func (n *CrossSignalNode) ID() string       { return n.id }
func (n *CrossSignalNode) NodeType() string { return "cross_signal" }
func (n *CrossSignalNode) Category() string { return "signal" }

func (n *CrossSignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "fast", Type: workflow.PortSeries, Required: true},
		{Name: "slow", Type: workflow.PortSeries, Required: true},
	}
}

func (n *CrossSignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "signal", Type: workflow.PortSignal, Required: false},
	}
}

func (n *CrossSignalNode) ParamSchema() []workflow.ParamDef {
	return nil
}

func (n *CrossSignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	fast, ok := toFloat64Slice(inputs["fast"])
	if !ok {
		return nil, fmt.Errorf("cross_signal: fast input must be []float64")
	}
	slow, ok := toFloat64Slice(inputs["slow"])
	if !ok {
		return nil, fmt.Errorf("cross_signal: slow input must be []float64")
	}
	if len(fast) != len(slow) {
		return nil, fmt.Errorf("cross_signal: fast(%d) and slow(%d) must have same length", len(fast), len(slow))
	}

	var signals []SignalEvent
	for i := 1; i < len(fast); i++ {
		prevFast, prevSlow := fast[i-1], slow[i-1]
		currFast, currSlow := fast[i], slow[i]
		prevAbove := prevFast > prevSlow
		currAbove := currFast > currSlow

		if !prevAbove && currAbove {
			signals = append(signals, SignalEvent{
				Index: i, Direction: "buy",
				FastValue: currFast, SlowValue: currSlow,
				Confidence: (currFast - currSlow) / currSlow,
			})
		} else if prevAbove && !currAbove {
			signals = append(signals, SignalEvent{
				Index: i, Direction: "sell",
				FastValue: currFast, SlowValue: currSlow,
				Confidence: (currSlow - currFast) / currSlow,
			})
		}
	}
	return map[string]any{"signal": signals}, nil
}

func (n *CrossSignalNode) Validate() error { return nil }
```

Create `app/internal/workflow/nodes/cross_signal_test.go`:

```go
package nodes

import (
	"context"
	"testing"
)

func TestCrossSignalNode_GoldenCross(t *testing.T) {
	node, _ := NewCrossSignalNode("cs", nil)
	fast := []float64{1, 2, 3, 5, 7}
	slow := []float64{2, 2, 2, 3, 4}

	outputs, err := node.Execute(context.Background(), map[string]any{"fast": fast, "slow": slow}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signals, ok := outputs["signal"].([]SignalEvent)
	if !ok {
		t.Fatalf("signal is %T", outputs["signal"])
	}
	if len(signals) != 1 {
		t.Fatalf("len(signals) = %d, want 1", len(signals))
	}
	if signals[0].Direction != "buy" {
		t.Errorf("direction = %q, want %q", signals[0].Direction, "buy")
	}
}

func TestCrossSignalNode_DeathCross(t *testing.T) {
	node, _ := NewCrossSignalNode("cs", nil)
	fast := []float64{7, 5, 3, 2, 1}
	slow := []float64{4, 4, 4, 3, 2}

	outputs, err := node.Execute(context.Background(), map[string]any{"fast": fast, "slow": slow}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	signals, ok := outputs["signal"].([]SignalEvent)
	if !ok {
		t.Fatalf("signal is %T", outputs["signal"])
	}
	if len(signals) == 0 {
		t.Fatal("expected at least one sell signal")
	}
	if signals[0].Direction != "sell" {
		t.Errorf("direction = %q, want %q", signals[0].Direction, "sell")
	}
}
```

- [ ] **Step 2: Implement LogOutputNode**

Create `app/internal/workflow/nodes/log_output.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/workflow"
)

// LogOutputNode logs its inputs and passes them through unchanged.
type LogOutputNode struct {
	id     string
	params map[string]any
}

// NewLogOutputNode creates a new LogOutputNode.
func NewLogOutputNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &LogOutputNode{id: id, params: params}, nil
}

func (n *LogOutputNode) ID() string       { return n.id }
func (n *LogOutputNode) NodeType() string { return "log_output" }
func (n *LogOutputNode) Category() string { return "output" }

func (n *LogOutputNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "input", Type: workflow.PortAny, Required: true},
	}
}

func (n *LogOutputNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "output", Type: workflow.PortAny, Required: false},
	}
}

func (n *LogOutputNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "prefix", Type: "string", Default: "", Description: "Log message prefix"},
	}
}

func (n *LogOutputNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	prefix := ""
	if p, ok := n.params["prefix"]; ok {
		prefix = fmt.Sprint(p)
	}
	if p, ok := params["prefix"]; ok {
		prefix = fmt.Sprint(p)
	}

	for k, v := range inputs {
		slog.Info(fmt.Sprintf("%s%s", prefix, k), "value", fmt.Sprint(v))
	}
	return inputs, nil
}

func (n *LogOutputNode) Validate() error { return nil }
```

Create `app/internal/workflow/nodes/log_output_test.go`:

```go
package nodes

import (
	"context"
	"testing"
)

func TestLogOutputNode_PassThrough(t *testing.T) {
	node, _ := NewLogOutputNode("log1", nil)
	inputs := map[string]any{"input": "hello"}
	outputs, err := node.Execute(context.Background(), inputs, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["input"] != "hello" {
		t.Errorf("pass-through failed: %v", outputs["input"])
	}
}
```

- [ ] **Step 3: Implement LoopNode**

Create `app/internal/workflow/nodes/loop.go`:

```go
package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// LoopNode iterates over an array input and emits each element on its output port.
type LoopNode struct {
	id     string
	params map[string]any
}

// NewLoopNode creates a new LoopNode.
func NewLoopNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &LoopNode{id: id, params: params}, nil
}

func (n *LoopNode) ID() string       { return n.id }
func (n *LoopNode) NodeType() string { return "loop" }
func (n *LoopNode) Category() string { return "control" }

func (n *LoopNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "items", Type: workflow.PortAny, Required: true},
	}
}

func (n *LoopNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "batched", Type: workflow.PortAny, Required: false},
	}
}

func (n *LoopNode) ParamSchema() []workflow.ParamDef {
	return nil
}

func (n *LoopNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	raw, ok := inputs["items"]
	if !ok {
		return nil, fmt.Errorf("loop: missing required input 'items'")
	}

	switch items := raw.(type) {
	case []any:
		return map[string]any{"batched": items}, nil
	case []string:
		result := make([]any, len(items))
		for i, s := range items {
			result[i] = s
		}
		return map[string]any{"batched": result}, nil
	default:
		return nil, fmt.Errorf("loop: items must be an array, got %T", raw)
	}
}

func (n *LoopNode) Validate() error { return nil }
```

Create `app/internal/workflow/nodes/loop_test.go`:

```go
package nodes

import (
	"context"
	"testing"
)

func TestLoopNode_StringArray(t *testing.T) {
	node, _ := NewLoopNode("loop1", nil)
	outputs, err := node.Execute(context.Background(), map[string]any{"items": []string{"a", "b", "c"}}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	batched, ok := outputs["batched"].([]any)
	if !ok {
		t.Fatalf("batched is %T", outputs["batched"])
	}
	if len(batched) != 3 {
		t.Fatalf("len = %d, want 3", len(batched))
	}
}

func TestLoopNode_AnyArray(t *testing.T) {
	node, _ := NewLoopNode("loop2", nil)
	outputs, err := node.Execute(context.Background(), map[string]any{"items": []any{1, "two", 3.0}}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	batched, ok := outputs["batched"].([]any)
	if !ok {
		t.Fatalf("batched is %T", outputs["batched"])
	}
	if len(batched) != 3 {
		t.Fatalf("len = %d, want 3", len(batched))
	}
}
```

- [ ] **Step 4: Run all node tests**

```bash
cd app && go test ./internal/workflow/nodes/ -v
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/workflow/nodes/cross_signal.go app/internal/workflow/nodes/cross_signal_test.go app/internal/workflow/nodes/log_output.go app/internal/workflow/nodes/log_output_test.go app/internal/workflow/nodes/loop.go app/internal/workflow/nodes/loop_test.go
git commit -m "feat(m2): implement CrossSignal, LogOutput, and Loop nodes"
```

---

## Milestone 3: DAG 执行引擎

### Task 13: Workflow/NodeInstance/Edge types + JSON serialization

**Files:**
- Create: `app/internal/workflow/dag.go`
- Create: `app/internal/workflow/dag_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/workflow/dag_test.go`:

```go
package workflow

import (
	"encoding/json"
	"testing"
)

func TestWorkflow_ParseJSON(t *testing.T) {
	input := `{
		"id": "wf1",
		"name": "test workflow",
		"nodes": [
			{"id": "n1", "node_type": "data_loader", "params": {"path": "data.csv"}},
			{"id": "n2", "node_type": "sma", "params": {"period": 10}}
		],
		"edges": [
			{"from_node": "n1", "from_port": "ohlcv", "to_node": "n2", "to_port": "input"}
		]
	}`
	var wf Workflow
	if err := json.Unmarshal([]byte(input), &wf); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if wf.ID != "wf1" {
		t.Errorf("ID = %q, want %q", wf.ID, "wf1")
	}
	if len(wf.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(wf.Nodes))
	}
	if wf.Nodes[0].NodeType != "data_loader" {
		t.Errorf("Nodes[0].NodeType = %q", wf.Nodes[0].NodeType)
	}
	if len(wf.Edges) != 1 {
		t.Fatalf("len(Edges) = %d, want 1", len(wf.Edges))
	}
	if wf.Edges[0].FromNode != "n1" {
		t.Errorf("Edges[0].FromNode = %q, want %q", wf.Edges[0].FromNode, "n1")
	}
}

func TestWorkflow_RoundTripJSON(t *testing.T) {
	wf := Workflow{
		ID:   "wf1",
		Name: "test",
		Nodes: []NodeInstance{
			{ID: "n1", NodeType: "data_loader", Params: map[string]any{"path": "data.csv"}},
		},
		Edges: []Edge{
			{FromNode: "n1", FromPort: "ohlcv", ToNode: "n2", ToPort: "input"},
		},
	}
	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	var wf2 Workflow
	if err := json.Unmarshal(data, &wf2); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if wf2.ID != wf.ID || wf2.Name != wf.Name {
		t.Error("round-trip mismatch")
	}
}

func TestTopoSort_SimplePipeline(t *testing.T) {
	wf := &Workflow{
		ID: "pipeline",
		Nodes: []NodeInstance{
			{ID: "a", NodeType: "passthrough"},
			{ID: "b", NodeType: "passthrough"},
			{ID: "c", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "a", FromPort: "out", ToNode: "b", ToPort: "in"},
			{FromNode: "b", FromPort: "out", ToNode: "c", ToPort: "in"},
		},
	}
	layers, err := TopoSort(wf)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	if len(layers) != 3 {
		t.Errorf("len(layers) = %d, want 3", len(layers))
	}
}

func TestTopoSort_Parallel(t *testing.T) {
	wf := &Workflow{
		ID: "fan-out",
		Nodes: []NodeInstance{
			{ID: "src", NodeType: "passthrough"},
			{ID: "a", NodeType: "passthrough"},
			{ID: "b", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "src", FromPort: "out", ToNode: "a", ToPort: "in"},
			{FromNode: "src", FromPort: "out", ToNode: "b", ToPort: "in"},
		},
	}
	layers, err := TopoSort(wf)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	// a and b should be in the same layer (parallel)
	if len(layers) != 2 {
		t.Errorf("len(layers) = %d, want 2 (source layer + parallel layer)", len(layers))
	}
	layer2 := layers[1]
	if len(layer2) != 2 {
		t.Errorf("layer2 len = %d, want 2 parallel nodes", len(layer2))
	}
}

func TestTopoSort_Cycle(t *testing.T) {
	wf := &Workflow{
		ID: "cycle",
		Nodes: []NodeInstance{
			{ID: "a", NodeType: "passthrough"},
			{ID: "b", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "a", FromPort: "out", ToNode: "b", ToPort: "in"},
			{FromNode: "b", FromPort: "out", ToNode: "a", ToPort: "in"},
		},
	}
	_, err := TopoSort(wf)
	if err == nil {
		t.Error("expected cycle error")
	}
}

func TestValidate_EmptyID(t *testing.T) {
	wf := &Workflow{ID: ""}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestValidate_NoNodes(t *testing.T) {
	wf := &Workflow{ID: "test"}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for no nodes")
	}
}

func TestValidate_DuplicateNodeID(t *testing.T) {
	wf := &Workflow{
		ID: "test",
		Nodes: []NodeInstance{
			{ID: "dup", NodeType: "sma"},
			{ID: "dup", NodeType: "sma"},
		},
	}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for duplicate node ID")
	}
}

func TestValidate_UnknownEdgeEndpoint(t *testing.T) {
	wf := &Workflow{
		ID: "test",
		Nodes: []NodeInstance{
			{ID: "a", NodeType: "sma"},
		},
		Edges: []Edge{
			{FromNode: "a", FromPort: "out", ToNode: "ghost", ToPort: "in"},
		},
	}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for unknown edge endpoint")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/workflow/ -v -run "TestWorkflow|TestTopo|TestValidate"`
Expected: FAIL — `Workflow`, `NodeInstance`, `Edge`, `TopoSort`, `Validate` not defined

- [ ] **Step 3: Implement DAG types**

Create `app/internal/workflow/dag.go`:

```go
package workflow

// Edge represents a connection between two node ports.
type Edge struct {
	FromNode string `json:"from_node"`
	FromPort string `json:"from_port"`
	ToNode   string `json:"to_node"`
	ToPort   string `json:"to_port"`
}

// NodeInstance is a concrete instance of a node type in a workflow graph.
type NodeInstance struct {
	ID       string         `json:"id"`
	NodeType string         `json:"node_type"`
	Params   map[string]any `json:"params,omitempty"`
}

// Workflow represents an executable DAG of node instances and edges.
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Nodes       []NodeInstance `json:"nodes"`
	Edges       []Edge         `json:"edges"`
}

// TopoLayer is a group of node IDs that can execute in parallel.
type TopoLayer []string

// TopoSort returns layers of node IDs ordered by Kahn's algorithm.
// Returns an error if the graph contains a cycle.
func TopoSort(wf *Workflow) ([]TopoLayer, error) {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	for _, n := range wf.Nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range wf.Edges {
		adj[e.FromNode] = append(adj[e.FromNode], e.ToNode)
		inDegree[e.ToNode]++
	}

	var queue []string
	for _, n := range wf.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var layers []TopoLayer
	for len(queue) > 0 {
		layer := make(TopoLayer, len(queue))
		copy(layer, queue)
		layers = append(layers, layer)

		var next []string
		for _, nodeID := range queue {
			for _, neighbor := range adj[nodeID] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					next = append(next, neighbor)
				}
			}
		}
		queue = next
	}

	for _, deg := range inDegree {
		if deg > 0 {
			return nil, &CycleError{Message: "workflow contains a cycle"}
		}
	}

	return layers, nil
}

// CycleError indicates the workflow graph has a cycle and cannot be executed.
type CycleError struct {
	Message string
}

func (e *CycleError) Error() string { return e.Message }

// Validate checks the workflow for structural correctness.
func Validate(wf *Workflow) error {
	if wf.ID == "" {
		return &ValidationError{Message: "workflow id is required"}
	}
	if len(wf.Nodes) == 0 {
		return &ValidationError{Message: "workflow must have at least one node"}
	}

	nodeIDs := make(map[string]bool)
	for _, n := range wf.Nodes {
		if n.ID == "" {
			return &ValidationError{Message: "node id is required"}
		}
		if n.NodeType == "" {
			return &ValidationError{Message: "node type is required for node " + n.ID}
		}
		if nodeIDs[n.ID] {
			return &ValidationError{Message: "duplicate node id: " + n.ID}
		}
		nodeIDs[n.ID] = true
	}

	for _, e := range wf.Edges {
		if !nodeIDs[e.FromNode] {
			return &ValidationError{Message: "edge from unknown node: " + e.FromNode}
		}
		if !nodeIDs[e.ToNode] {
			return &ValidationError{Message: "edge to unknown node: " + e.ToNode}
		}
	}

	if _, err := TopoSort(wf); err != nil {
		return err
	}

	return nil
}

// ValidationError indicates a structural problem with the workflow definition.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/workflow/ -v -run "TestWorkflow|TestTopo|TestValidate"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/workflow/dag.go app/internal/workflow/dag_test.go
git commit -m "feat(m3): add Workflow types, TopoSort, and Validate with cycle detection"
```

---

### Task 14: LRU Cache

**Files:**
- Create: `app/internal/workflow/cache.go`

- [ ] **Step 1: Implement cache wrapper**

Create `app/internal/workflow/cache.go`:

```go
package workflow

import (
	"crypto/sha256"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
)

const defaultCacheSize = 256

// NodeCache provides LRU caching for node execution results.
type NodeCache struct {
	cache *lru.Cache[string, map[string]any]
}

// NewNodeCache creates a cache with the given max size.
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

// CacheKey generates a deterministic key from node ID and inputs.
func CacheKey(nodeID string, inputs map[string]any) string {
	data := fmt.Sprintf("%s:%v", nodeID, inputs)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:16])
}

// Get returns cached output, or nil if not present.
func (c *NodeCache) Get(key string) (map[string]any, bool) {
	return c.cache.Get(key)
}

// Put stores output in the cache.
func (c *NodeCache) Put(key string, outputs map[string]any) {
	c.cache.Add(key, outputs)
}

// Len returns the current number of entries.
func (c *NodeCache) Len() int {
	return c.cache.Len()
}
```

- [ ] **Step 2: Install lru dependency**

```bash
cd app && go get github.com/hashicorp/golang-lru/v2
```

- [ ] **Step 3: Verify build**

```bash
cd app && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add app/internal/workflow/cache.go app/go.mod app/go.sum
git commit -m "feat(m3): add LRU cache for node execution results"
```

---

### Task 15: Engine.Execute

**Files:**
- Create: `app/internal/workflow/engine.go`
- Create: `app/internal/workflow/engine_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/workflow/engine_test.go`:

```go
package workflow

import (
	"context"
	"testing"
	"time"
)

func TestEngine_ExecuteSimpleDAG(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, err := NewEngine(reg, 64)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	wf := &Workflow{
		ID:   "test",
		Name: "simple pipeline",
		Nodes: []NodeInstance{
			{ID: "n1", NodeType: "passthrough", Params: map[string]any{"value": "hello"}},
			{ID: "n2", NodeType: "passthrough", Params: map[string]any{}},
		},
		Edges: []Edge{
			{FromNode: "n1", FromPort: "output", ToNode: "n2", ToPort: "input"},
		},
	}

	result, err := engine.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if len(result.NodeResults) != 2 {
		t.Errorf("len(NodeResults) = %d, want 2", len(result.NodeResults))
	}
}

func TestEngine_ExecuteWithTimeout(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithCategory("slow", func(id string, params map[string]any) (BaseNode, error) {
		return &slowNode{id: id}, nil
	}, "test")

	engine, _ := NewEngine(reg, 64)

	wf := &Workflow{
		ID: "timeout_test",
		Nodes: []NodeInstance{
			{ID: "s1", NodeType: "slow", Params: nil},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := engine.Execute(ctx, wf)
	if err == nil {
		t.Error("expected error due to timeout")
	}
}

func TestEngine_ExecuteParallelDAG(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, _ := NewEngine(reg, 64)

	wf := &Workflow{
		ID: "parallel",
		Nodes: []NodeInstance{
			{ID: "src", NodeType: "passthrough", Params: map[string]any{"value": "data"}},
			{ID: "w1", NodeType: "passthrough"},
			{ID: "w2", NodeType: "passthrough"},
			{ID: "snk", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "src", FromPort: "output", ToNode: "w1", ToPort: "input"},
			{FromNode: "src", FromPort: "output", ToNode: "w2", ToPort: "input"},
			{FromNode: "w1", FromPort: "output", ToNode: "snk", ToPort: "input"},
			{FromNode: "w2", FromPort: "output", ToNode: "snk", ToPort: "input"},
		},
	}

	result, err := engine.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if len(result.NodeResults) != 4 {
		t.Errorf("len(NodeResults) = %d, want 4", len(result.NodeResults))
	}
}

// passthroughNode copies its input (or param value) to its output.
type passthroughNode struct {
	id     string
	params map[string]any
}

func (n *passthroughNode) ID() string                   { return n.id }
func (n *passthroughNode) NodeType() string             { return "passthrough" }
func (n *passthroughNode) Category() string             { return "test" }
func (n *passthroughNode) InputPorts() []PortDefinition  { return []PortDefinition{{Name: "input", Type: PortAny, Required: false}} }
func (n *passthroughNode) OutputPorts() []PortDefinition { return []PortDefinition{{Name: "output", Type: PortAny, Required: false}} }
func (n *passthroughNode) ParamSchema() []ParamDef       { return nil }
func (n *passthroughNode) Validate() error               { return nil }
func (n *passthroughNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	if v, ok := params["value"]; ok {
		return map[string]any{"output": v}, nil
	}
	if v, ok := inputs["input"]; ok {
		return map[string]any{"output": v}, nil
	}
	return map[string]any{"output": "default"}, nil
}

// slowNode sleeps then returns — used for timeout tests.
type slowNode struct{ id string }

func (n *slowNode) ID() string                   { return n.id }
func (n *slowNode) NodeType() string             { return "slow" }
func (n *slowNode) Category() string             { return "test" }
func (n *slowNode) InputPorts() []PortDefinition  { return nil }
func (n *slowNode) OutputPorts() []PortDefinition { return nil }
func (n *slowNode) ParamSchema() []ParamDef       { return nil }
func (n *slowNode) Validate() error               { return nil }
func (n *slowNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(1 * time.Second):
	}
	return nil, nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/workflow/ -v -run TestEngine`
Expected: FAIL — `Engine`, `NewEngine` not defined

- [ ] **Step 3: Implement Engine**

Create `app/internal/workflow/engine.go`:

```go
package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ExecutionResult holds the outcome of a workflow run.
type ExecutionResult struct {
	WorkflowID  string       `json:"workflow_id"`
	Status      string       `json:"status"` // "success" | "failed" | "partial"
	StartedAt   time.Time    `json:"started_at"`
	FinishedAt  time.Time    `json:"finished_at"`
	NodeResults []NodeResult `json:"node_results"`
	Error       string       `json:"error,omitempty"`
}

// NodeResult holds the result of a single node execution.
type NodeResult struct {
	NodeID   string         `json:"node_id"`
	NodeType string         `json:"node_type"`
	Status   string         `json:"status"` // "success" | "failed" | "skipped"
	Duration time.Duration  `json:"duration"`
	Outputs  map[string]any `json:"outputs,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// Engine executes workflow DAGs.
type Engine struct {
	registry *NodeRegistry
	cache    *NodeCache
}

// NewEngine creates a new workflow execution engine.
func NewEngine(registry *NodeRegistry, cacheSize int) (*Engine, error) {
	cache, err := NewNodeCache(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}
	return &Engine{registry: registry, cache: cache}, nil
}

// Execute runs a workflow to completion.
func (e *Engine) Execute(ctx context.Context, wf *Workflow) (*ExecutionResult, error) {
	result := &ExecutionResult{
		WorkflowID: wf.ID,
		StartedAt:  time.Now(),
	}
	defer func() { result.FinishedAt = time.Now() }()

	// 1. Validate structure
	if err := Validate(wf); err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	// 2. Topo sort into layers
	layers, err := TopoSort(wf)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	slog.Info("executing workflow",
		"id", wf.ID,
		"name", wf.Name,
		"layers", len(layers),
		"nodes", len(wf.Nodes))

	// 3. Execute layer by layer
	upstreamOutputs := &sync.Map{}
	nodeResults := make([]NodeResult, len(wf.Nodes))
	nodeResultByID := make(map[string]*NodeResult)
	for i, n := range wf.Nodes {
		nodeResults[i] = NodeResult{NodeID: n.ID, NodeType: n.NodeType}
		nodeResultByID[n.ID] = &nodeResults[i]
	}

	for layerIdx, layer := range layers {
		g, layerCtx := errgroup.WithContext(ctx)

		for _, nodeID := range layer {
			nodeID := nodeID
			layerIdx := layerIdx
			g.Go(func() error {
				return e.executeNode(layerCtx, wf, nodeID, layerIdx, upstreamOutputs, nodeResultByID)
			})
		}

		if err := g.Wait(); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.NodeResults = nodeResults
			return result, err
		}
	}

	result.Status = "success"
	result.NodeResults = nodeResults
	return result, nil
}

func (e *Engine) executeNode(
	ctx context.Context,
	wf *Workflow,
	nodeID string,
	layerIdx int,
	upstreamOutputs *sync.Map,
	nodeResultByID map[string]*NodeResult,
) error {
	var nodeInstance *NodeInstance
	for i := range wf.Nodes {
		if wf.Nodes[i].ID == nodeID {
			nodeInstance = &wf.Nodes[i]
			break
		}
	}
	if nodeInstance == nil {
		return fmt.Errorf("node %q not found in workflow", nodeID)
	}

	nr := nodeResultByID[nodeID]
	start := time.Now()

	node, err := e.registry.Create(nodeInstance.NodeType, nodeInstance.ID, nodeInstance.Params)
	if err != nil {
		nr.Status = "failed"
		nr.Error = err.Error()
		nr.Duration = time.Since(start)
		return fmt.Errorf("create node %q: %w", nodeID, err)
	}

	// Collect inputs from upstream edges
	inputs := make(map[string]any)
	for _, edge := range wf.Edges {
		if edge.ToNode == nodeID {
			if val, ok := upstreamOutputs.Load(edge.FromNode); ok {
				outputs := val.(map[string]any)
				if v, ok := outputs[edge.FromPort]; ok {
					inputs[edge.ToPort] = v
				}
			}
		}
	}

	// Merge instance params for source nodes
	if len(inputs) == 0 && nodeInstance.Params != nil {
		for k, v := range nodeInstance.Params {
			inputs[k] = v
		}
	}

	// Check cache
	cacheKey := CacheKey(nodeID, inputs)
	if cached, ok := e.cache.Get(cacheKey); ok {
		nr.Status = "success"
		nr.Outputs = cached
		nr.Duration = time.Since(start)
		slog.Debug("cache hit", "node", nodeID, "layer", layerIdx)
		upstreamOutputs.Store(nodeID, cached)
		return nil
	}

	outputs, err := node.Execute(ctx, inputs, nil)
	nr.Duration = time.Since(start)

	if err != nil {
		nr.Status = "failed"
		nr.Error = err.Error()
		return fmt.Errorf("execute node %q: %w", nodeID, err)
	}

	nr.Status = "success"
	nr.Outputs = outputs
	upstreamOutputs.Store(nodeID, outputs)
	e.cache.Put(cacheKey, outputs)

	slog.Debug("node executed", "node", nodeID, "type", nodeInstance.NodeType, "duration", nr.Duration)
	return nil
}
```

- [ ] **Step 4: Install errgroup dependency**

```bash
cd app && go get golang.org/x/sync/errgroup
```

- [ ] **Step 5: Run engine tests**

Run: `cd app && go test ./internal/workflow/ -v -run TestEngine`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/internal/workflow/engine.go app/internal/workflow/engine_test.go app/go.mod app/go.sum
git commit -m "feat(m3): implement Engine with DAG layer-parallel execution and caching"
```

---

### Task 16: CLI runner (minimal)

**Files:**
- Create: `cmd/qf/main.go`
- Create: `examples/sma_cross.json`
- Create: `examples/aapl_sample.csv`

- [ ] **Step 1: Implement CLI**

Create `cmd/qf/main.go`:

```go
// qf is the QuantFlow CLI — run workflows, list nodes, validate graphs.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/storage"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: qf <command> [args]\n")
		fmt.Fprintf(os.Stderr, "  qf run <file>         Execute a workflow from JSON file\n")
		fmt.Fprintf(os.Stderr, "  qf run --id <uuid>    Execute a saved workflow by ID\n")
		fmt.Fprintf(os.Stderr, "  qf nodes              List registered node types\n")
		fmt.Fprintf(os.Stderr, "  qf validate <file>    Validate a workflow JSON\n")
		fmt.Fprintf(os.Stderr, "  qf version            Print version\n")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "version":
		cfg, _ := config.Load()
		fmt.Println(cfg.Version)
	case "nodes":
		cmdNodes()
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: qf validate <file>")
			os.Exit(1)
		}
		cmdValidate(os.Args[2])
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: qf run <file>")
			os.Exit(1)
		}
		cmdRun(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func loadWorkflow(path string) (*workflow.Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var wf workflow.Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return &wf, nil
}

func cmdValidate(path string) {
	wf, err := loadWorkflow(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := workflow.Validate(wf); err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Valid.")
}

func cmdNodes() {
	r := newRegistry()
	all := r.ListAll()
	byCat := make(map[string][]string)
	for _, m := range all {
		byCat[m.Category] = append(byCat[m.Category], m.NodeType)
	}
	categories := []string{"data", "indicator", "signal", "output", "control", "test"}
	for _, cat := range categories {
		types, ok := byCat[cat]
		if !ok {
			continue
		}
		fmt.Printf("[%s]\n", cat)
		for _, t := range types {
			fmt.Printf("  %s\n", t)
		}
	}
}

func cmdRun(arg string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	logging.Setup(cfg.LogLevel)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrations, err := storage.BuiltinMigrations()
	if err != nil {
		slog.Error("failed to load migrations", "error", err)
		os.Exit(1)
	}
	if err := storage.Run(db, migrations); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	var wf *workflow.Workflow
	repo := storage.NewWorkflowRepo(db)

	if arg == "--id" {
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: qf run --id <uuid>")
			os.Exit(1)
		}
		wfID := os.Args[3]
		var err error
		wf, err = repo.Load(wfID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Load error: %v\n", err)
			os.Exit(1)
		}
	} else {
		wf, err = loadWorkflow(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	reg := newRegistry()
	engine, err := workflow.NewEngine(reg, 256)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine init error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	result, err := engine.Execute(ctx, wf)

	if err := repo.SaveExecution(wf.ID, 0, result.Status, result); err != nil {
		slog.Warn("failed to save execution history", "error", err)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		os.Exit(1)
	}
	for _, nr := range result.NodeResults {
		status := "✓"
		if nr.Status != "success" {
			status = "✗"
		}
		fmt.Printf("[qf]   [%s:%s] %s (%v)\n", nr.NodeType, nr.NodeID, status, nr.Duration)
	}
	fmt.Printf("[qf] Done. %s\n", result.Status)
}

func newRegistry() *workflow.NodeRegistry {
	r := workflow.NewRegistry()
	r.RegisterWithCategory("data_loader", nodes.NewDataLoaderNode, "data")
	r.RegisterWithCategory("sma", nodes.NewSMANode, "indicator")
	r.RegisterWithCategory("cross_signal", nodes.NewCrossSignalNode, "signal")
	r.RegisterWithCategory("log_output", nodes.NewLogOutputNode, "output")
	r.RegisterWithCategory("loop", nodes.NewLoopNode, "control")
	return r
}
```

- [ ] **Step 2: Create test data and example workflow**

Create `examples/aapl_sample.csv`:

```csv
date,open,high,low,close,volume
2024-01-01,150,155,149,153,10000000
2024-01-02,153,158,152,157,12000000
2024-01-03,157,159,154,155,11000000
2024-01-04,155,160,153,159,13000000
2024-01-05,159,162,158,161,14000000
2024-01-08,161,165,160,163,12500000
2024-01-09,163,166,161,162,11500000
2024-01-10,162,167,160,165,13500000
2024-01-11,165,169,164,168,14500000
2024-01-12,168,172,167,171,15000000
2024-01-16,171,173,168,170,12800000
2024-01-17,170,174,169,172,13200000
2024-01-18,172,176,171,175,14000000
2024-01-19,175,178,174,177,13800000
2024-01-22,177,180,176,179,14200000
2024-01-23,179,181,177,178,13600000
2024-01-24,178,182,177,180,14400000
2024-01-25,180,183,179,182,14800000
2024-01-26,182,185,181,184,15000000
2024-01-29,184,186,182,185,14600000
2024-01-30,185,187,183,184,14000000
2024-01-31,184,186,181,183,13800000
```

Create `examples/sma_cross.json`:

```json
{
  "id": "sma-cross-demo",
  "name": "SMA Crossover Strategy",
  "nodes": [
    {
      "id": "load_aapl",
      "node_type": "data_loader",
      "params": {
        "source": "csv",
        "path": "examples/aapl_sample.csv"
      }
    },
    {
      "id": "ma_fast",
      "node_type": "sma",
      "params": { "period": 5 }
    },
    {
      "id": "ma_slow",
      "node_type": "sma",
      "params": { "period": 20 }
    },
    {
      "id": "signal",
      "node_type": "cross_signal",
      "params": {}
    },
    {
      "id": "print",
      "node_type": "log_output",
      "params": { "prefix": "[signal] " }
    }
  ],
  "edges": [
    { "from_node": "load_aapl", "from_port": "ohlcv", "to_node": "ma_fast", "to_port": "input" },
    { "from_node": "load_aapl", "from_port": "ohlcv", "to_node": "ma_slow", "to_port": "input" },
    { "from_node": "ma_fast", "from_port": "output", "to_node": "signal", "to_port": "fast" },
    { "from_node": "ma_slow", "from_port": "output", "to_node": "signal", "to_port": "slow" },
    { "from_node": "signal", "from_port": "signal", "to_node": "print", "to_port": "input" }
  ]
}
```

- [ ] **Step 3: Build and test CLI**

```bash
cd app && go build -o qf ./cmd/qf/ && ./qf run examples/sma_cross.json
```
Expected: Shows topo sort layers, node execution with ✓ marks and durations, "Done. success"

- [ ] **Step 4: Commit**

```bash
git add cmd/qf/main.go examples/sma_cross.json examples/aapl_sample.csv
git commit -m "feat(m3): add qf CLI with run/validate/nodes/version commands and SMA cross example"
```

---

## Milestone 4: 持久化与恢复

### Task 17: Migrations 002_workflows and 003_checkpoints

**Files:**
- Create: `app/internal/storage/migrations/002_workflows.sql`
- Create: `app/internal/storage/migrations/003_checkpoints.sql`

- [ ] **Step 1: Create workflow storage schema**

Create `app/internal/storage/migrations/002_workflows.sql`:

```sql
-- 002_workflows: workflow storage with versioning and execution history.

CREATE TABLE IF NOT EXISTS workflows (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS workflow_versions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    graph_json  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(workflow_id, version)
);

CREATE TABLE IF NOT EXISTS execution_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT NOT NULL,
    version     INTEGER NOT NULL DEFAULT 0,
    started_at  TEXT,
    finished_at TEXT,
    status      TEXT NOT NULL DEFAULT 'pending',
    result_json TEXT,
    error_msg   TEXT DEFAULT ''
);
```

Create `app/internal/storage/migrations/003_checkpoints.sql`:

```sql
-- 003_checkpoints: execution checkpoint table for breakpoint recovery.

CREATE TABLE IF NOT EXISTS execution_checkpoints (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id        TEXT NOT NULL,
    workflow_id   TEXT NOT NULL,
    node_id       TEXT NOT NULL,
    input_hash    TEXT NOT NULL,
    outputs_json  TEXT NOT NULL DEFAULT '{}',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_run_node
    ON execution_checkpoints(run_id, node_id);
```

- [ ] **Step 2: Verify migrations run**

```bash
cd app && go test ./internal/storage/ -v
```
Expected: PASS (new migrations picked up by embed)

- [ ] **Step 3: Commit**

```bash
git add app/internal/storage/migrations/002_workflows.sql app/internal/storage/migrations/003_checkpoints.sql
git commit -m "feat(m4): add workflow storage and checkpoint migration schemas"
```

---

### Task 18: WorkflowRepo

**Files:**
- Create: `app/internal/storage/workflow_repo.go`
- Create: `app/internal/storage/workflow_repo_test.go`

- [ ] **Step 1: Write the test**

Create `app/internal/storage/workflow_repo_test.go`:

```go
package storage

import (
	"testing"

	"quantflow/internal/workflow"
)

func setupRepo(t *testing.T) (*WorkflowRepo, func()) {
	t.Helper()
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	migrations, err := BuiltinMigrations()
	if err != nil {
		t.Fatalf("BuiltinMigrations() error = %v", err)
	}
	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	repo := NewWorkflowRepo(db)
	return repo, func() { db.Close() }
}

func TestWorkflowRepo_SaveAndLoad(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	wf := &workflow.Workflow{
		ID:   "wf-test-1",
		Name: "Test Workflow",
		Nodes: []workflow.NodeInstance{
			{ID: "n1", NodeType: "sma", Params: map[string]any{"period": float64(10)}},
		},
		Edges: []workflow.Edge{},
	}

	if err := repo.Save(wf); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := repo.Load("wf-test-1", nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Name != "Test Workflow" {
		t.Errorf("Name = %q, want %q", loaded.Name, "Test Workflow")
	}
	if len(loaded.Nodes) != 1 {
		t.Errorf("len(Nodes) = %d, want 1", len(loaded.Nodes))
	}
}

func TestWorkflowRepo_SaveCreatesNewVersion(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	wf := &workflow.Workflow{ID: "wf-v", Name: "V1"}
	if err := repo.Save(wf); err != nil {
		t.Fatalf("Save() v1 error = %v", err)
	}

	wf.Name = "V2"
	if err := repo.Save(wf); err != nil {
		t.Fatalf("Save() v2 error = %v", err)
	}

	latest, err := repo.Load("wf-v", nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if latest.Name != "V2" {
		t.Errorf("latest Name = %q, want %q", latest.Name, "V2")
	}

	v1 := 1
	v1wf, err := repo.Load("wf-v", &v1)
	if err != nil {
		t.Fatalf("Load(v1) error = %v", err)
	}
	if v1wf.Name != "V1" {
		t.Errorf("v1 Name = %q, want %q", v1wf.Name, "V1")
	}
}

func TestWorkflowRepo_List(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	wf1 := &workflow.Workflow{ID: "wf-list-a", Name: "WF A"}
	wf2 := &workflow.Workflow{ID: "wf-list-b", Name: "WF B"}
	wf3 := &workflow.Workflow{ID: "wf-list-c", Name: "WF C"}
	repo.Save(wf1)
	repo.Save(wf2)
	repo.Save(wf3)

	metas, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(metas) < 3 {
		t.Errorf("len(metas) = %d, want >= 3", len(metas))
	}
}

func TestWorkflowRepo_LoadNotFound(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	_, err := repo.Load("nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent workflow")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/storage/ -v -run TestWorkflowRepo`
Expected: FAIL — `WorkflowRepo` not defined

- [ ] **Step 3: Implement WorkflowRepo**

Create `app/internal/storage/workflow_repo.go`:

```go
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"quantflow/internal/workflow"
)

// WorkflowMeta holds basic info about a saved workflow.
type WorkflowMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

// WorkflowRepo provides CRUD operations for workflow persistence.
type WorkflowRepo struct {
	db *sql.DB
}

// NewWorkflowRepo creates a new WorkflowRepo.
func NewWorkflowRepo(db *sql.DB) *WorkflowRepo {
	return &WorkflowRepo{db: db}
}

// Save persists a workflow, creating a new version.
func (r *WorkflowRepo) Save(wf *workflow.Workflow) error {
	graphJSON, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal workflow: %w", err)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO workflows (id, name, description, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			updated_at = datetime('now')`,
		wf.ID, wf.Name, wf.Description)
	if err != nil {
		return fmt.Errorf("upsert workflow: %w", err)
	}

	var maxVersion int
	if err := tx.QueryRow("SELECT COALESCE(MAX(version), 0) FROM workflow_versions WHERE workflow_id = ?", wf.ID).Scan(&maxVersion); err != nil {
		return fmt.Errorf("get max version: %w", err)
	}
	nextVersion := maxVersion + 1

	_, err = tx.Exec(`INSERT INTO workflow_versions (workflow_id, version, graph_json)
		VALUES (?, ?, ?)`, wf.ID, nextVersion, string(graphJSON))
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}

	return tx.Commit()
}

// Load retrieves a workflow by ID. If version is nil, the latest is returned.
func (r *WorkflowRepo) Load(id string, version *int) (*workflow.Workflow, error) {
	var graphJSON string
	if version != nil {
		err := r.db.QueryRow(
			"SELECT graph_json FROM workflow_versions WHERE workflow_id = ? AND version = ?",
			id, *version,
		).Scan(&graphJSON)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow %q version %d not found", id, *version)
		}
		if err != nil {
			return nil, fmt.Errorf("load version: %w", err)
		}
	} else {
		err := r.db.QueryRow(
			`SELECT graph_json FROM workflow_versions
			 WHERE workflow_id = ?
			 ORDER BY version DESC LIMIT 1`, id,
		).Scan(&graphJSON)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow %q not found", id)
		}
		if err != nil {
			return nil, fmt.Errorf("load latest: %w", err)
		}
	}

	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(graphJSON), &wf); err != nil {
		return nil, fmt.Errorf("unmarshal workflow: %w", err)
	}
	return &wf, nil
}

// List returns metadata for all saved workflows.
func (r *WorkflowRepo) List() ([]WorkflowMeta, error) {
	rows, err := r.db.Query("SELECT id, name, description, updated_at FROM workflows ORDER BY updated_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	var metas []WorkflowMeta
	for rows.Next() {
		var m WorkflowMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// SaveExecution records the result of a workflow execution.
func (r *WorkflowRepo) SaveExecution(workflowID string, version int, status string, result *workflow.ExecutionResult) error {
	var resultJSON string
	var errMsg string
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		resultJSON = string(data)
		if result.Error != "" {
			errMsg = result.Error
		}
	}

	startedAt := time.Now().Format(time.RFC3339)
	finishedAt := time.Now().Format(time.RFC3339)

	_, err := r.db.Exec(`INSERT INTO execution_history
		(workflow_id, version, started_at, finished_at, status, result_json, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		workflowID, version, startedAt, finishedAt, status, resultJSON, errMsg)
	if err != nil {
		return fmt.Errorf("save execution: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test**

Run: `cd app && go test ./internal/storage/ -v -run TestWorkflowRepo`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/internal/storage/workflow_repo.go app/internal/storage/workflow_repo_test.go
git commit -m "feat(m4): add WorkflowRepo with Save/Load/List/versioning"
```

---

## Milestone 5: CLI 工具 + 测试覆盖

### Task 19: Remaining example workflows

**Files:**
- Create: `examples/multi_asset.json`
- Create: `examples/error_handling.json`

- [ ] **Step 1: Create multi_asset.json**

Create `examples/multi_asset.json`:

```json
{
  "id": "multi-asset-demo",
  "name": "Multi-Asset SMA Scanner",
  "description": "Runs SMA crossover detection on multiple symbols using a loop node",
  "nodes": [
    {
      "id": "symbols",
      "node_type": "loop",
      "params": {}
    },
    {
      "id": "indicator",
      "node_type": "sma",
      "params": { "period": 10 }
    },
    {
      "id": "output",
      "node_type": "log_output",
      "params": { "prefix": "[scanner] " }
    }
  ],
  "edges": [
    { "from_node": "symbols", "from_port": "batched", "to_node": "indicator", "to_port": "input" },
    { "from_node": "indicator", "from_port": "output", "to_node": "output", "to_port": "input" }
  ]
}
```

- [ ] **Step 2: Create error_handling.json**

Create `examples/error_handling.json`:

```json
{
  "id": "error-demo",
  "name": "Error Handling Demo",
  "description": "Demonstrates graceful failure when a node gets bad input",
  "nodes": [
    {
      "id": "load_missing",
      "node_type": "data_loader",
      "params": {
        "source": "csv",
        "path": "examples/nonexistent_file.csv"
      }
    },
    {
      "id": "sma",
      "node_type": "sma",
      "params": { "period": 20 }
    },
    {
      "id": "print",
      "node_type": "log_output",
      "params": {}
    }
  ],
  "edges": [
    { "from_node": "load_missing", "from_port": "ohlcv", "to_node": "sma", "to_port": "input" },
    { "from_node": "sma", "from_port": "output", "to_node": "print", "to_port": "input" }
  ]
}
```

- [ ] **Step 3: Commit**

```bash
git add examples/multi_asset.json examples/error_handling.json
git commit -m "feat(m5): add multi-asset and error handling example workflows"
```

---

### Task 20: Benchmark tests

**Files:**
- Create: `app/internal/workflow/bench_test.go`

- [ ] **Step 1: Write benchmarks**

Create `app/internal/workflow/bench_test.go`:

```go
package workflow

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkEngine_100NodeDAG(b *testing.B) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, _ := NewEngine(reg, 512)

	nodes := make([]NodeInstance, 100)
	edges := make([]Edge, 0)
	for i := 0; i < 100; i++ {
		nodes[i] = NodeInstance{
			ID:       fmt.Sprintf("n%d", i+1),
			NodeType: "passthrough",
			Params:   map[string]any{"value": "data"},
		}
		if i > 0 {
			edges = append(edges, Edge{
				FromNode: fmt.Sprintf("n%d", i),
				FromPort: "output",
				ToNode:   fmt.Sprintf("n%d", i+1),
				ToPort:   "input",
			})
		}
	}

	wf := &Workflow{ID: "bench-100", Name: "100-node pipeline", Nodes: nodes, Edges: edges}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(context.Background(), wf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEngine_WideDAG(b *testing.B) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, _ := NewEngine(reg, 512)

	nodes := []NodeInstance{
		{ID: "source", NodeType: "passthrough", Params: map[string]any{"value": "data"}},
	}
	edges := make([]Edge, 0)

	for i := 0; i < 50; i++ {
		nodeID := fmt.Sprintf("worker%d", i)
		nodes = append(nodes, NodeInstance{ID: nodeID, NodeType: "passthrough"})
		edges = append(edges, Edge{
			FromNode: "source", FromPort: "output",
			ToNode: nodeID, ToPort: "input",
		})
	}

	nodes = append(nodes, NodeInstance{ID: "sink", NodeType: "passthrough"})
	for i := 0; i < 50; i++ {
		edges = append(edges, Edge{
			FromNode: fmt.Sprintf("worker%d", i), FromPort: "output",
			ToNode: "sink", ToPort: "input",
		})
	}

	wf := &Workflow{ID: "bench-wide", Name: "wide fan-out", Nodes: nodes, Edges: edges}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(context.Background(), wf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
```

- [ ] **Step 2: Run benchmarks**

```bash
cd app && go test ./internal/workflow/ -bench=. -benchmem
```

- [ ] **Step 3: Commit**

```bash
git add app/internal/workflow/bench_test.go
git commit -m "feat(m5): add 100-node and wide DAG benchmarks"
```

---

### Task 21: Integration test

**Files:**
- Create: `app/internal/workflow/integration_test.go`

- [ ] **Step 1: Write integration test**

Create `app/internal/workflow/integration_test.go`:

```go
//go:build integration
// +build integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"quantflow/internal/workflow/nodes"
)

func TestIntegration_SMACrossFromFile(t *testing.T) {
	tmp := t.TempDir()
	csvPath := filepath.Join(tmp, "data.csv")
	os.WriteFile(csvPath, []byte(`date,open,high,low,close,volume
2024-01-01,100,110,95,105,1000
2024-01-02,105,115,100,110,1200
2024-01-03,110,120,105,115,1300
2024-01-04,115,125,110,120,1400
2024-01-05,120,130,115,125,1500`), 0644)

	reg := NewRegistry()
	reg.RegisterWithCategory("data_loader", nodes.NewDataLoaderNode, "data")
	reg.RegisterWithCategory("sma", nodes.NewSMANode, "indicator")
	reg.RegisterWithCategory("cross_signal", nodes.NewCrossSignalNode, "signal")
	reg.RegisterWithCategory("log_output", nodes.NewLogOutputNode, "output")

	engine, err := NewEngine(reg, 64)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	wf := &Workflow{
		ID:   "integration-test",
		Name: "SMA Cross Integration",
		Nodes: []NodeInstance{
			{ID: "loader", NodeType: "data_loader", Params: map[string]any{"source": "csv", "path": csvPath}},
			{ID: "fast", NodeType: "sma", Params: map[string]any{"period": 2}},
			{ID: "slow", NodeType: "sma", Params: map[string]any{"period": 3}},
			{ID: "signal", NodeType: "cross_signal"},
			{ID: "log", NodeType: "log_output"},
		},
		Edges: []Edge{
			{FromNode: "loader", FromPort: "ohlcv", ToNode: "fast", ToPort: "input"},
			{FromNode: "loader", FromPort: "ohlcv", ToNode: "slow", ToPort: "input"},
			{FromNode: "fast", FromPort: "output", ToNode: "signal", ToPort: "fast"},
			{FromNode: "slow", FromPort: "output", ToNode: "signal", ToPort: "slow"},
			{FromNode: "signal", FromPort: "signal", ToNode: "log", ToPort: "input"},
		},
	}

	result, err := engine.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	var signalResult *NodeResult
	for i := range result.NodeResults {
		if result.NodeResults[i].NodeID == "signal" {
			signalResult = &result.NodeResults[i]
			break
		}
	}
	if signalResult == nil {
		t.Fatal("missing signal node result")
	}
	if signalResult.Status != "success" {
		t.Errorf("signal status = %q, want %q", signalResult.Status, "success")
	}
}
```

- [ ] **Step 2: Run integration tests**

```bash
cd app && go test -tags=integration ./internal/workflow/ -v
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add app/internal/workflow/integration_test.go
git commit -m "test(m5): add integration test for full SMA cross pipeline"
```

---

### Task 22: Race detection and final verification

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Run race detector**

```bash
cd app && go test -race ./... -v -count=1
```

- [ ] **Step 2: Fix any race conditions**

If races are detected, add appropriate synchronization. Common fixes: `sync.Mutex` for shared maps, `atomic` for counters.

- [ ] **Step 3: Run coverage**

```bash
cd app && make coverage
```

- [ ] **Step 4: Update CHANGELOG**

Edit `CHANGELOG.md`, replace the `[Unreleased]` section:

```markdown
## [2026.6.16] - 2026-06-16

### Added
- [Engine] Engine-first Phase 1 implementation: pure-Go workflow engine
- [Engine] BaseNode interface with PortType system and NodeRegistry (M2)
- [Engine] DAG execution engine with Kahn topo sort, goroutine layer-parallel execution, and LRU caching (M3)
- [Engine] 5 built-in node types: DataLoader (CSV), SMA, CrossSignal, LogOutput, Loop (M2)
- [Storage] SQLite WAL database with embedded migration framework (M1)
- [Storage] WorkflowRepo with versioned persistence and execution history (M4)
- [Storage] Execution checkpoint table for breakpoint recovery (M4)
- [Frontend] qf CLI tool: run, nodes, validate, version commands (M3-M5)
- [Frontend] Example workflows: sma_cross, multi_asset, error_handling (M5)
- [Docs] Phase 1 Engine-First design doc and implementation plan
- [Docs] Benchmark suite for 100-node pipeline and wide DAG

### Changed
- [Docs] Phase 1 restructured from Proposal's 4-parallel-track to Engine-First serial milestones
```

- [ ] **Step 5: Final verification**

```bash
cd app && make build && make test && make lint
```
Expected: all green

- [ ] **Step 6: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for Phase 1 Engine-First completion"
```
