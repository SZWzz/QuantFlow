# Fix Python Sidecar Issues

## Motivation

Three issues in the Python sidecar:

1. **`requirements.lock` contains 198 packages** including `psycopg2-binary` (Postgres — violates "SQLite only" rule), `fastapi`/`uvicorn` (HTTP server — not needed for gRPC), `duckdb` (another DB engine), and `matplotlib`/`plotly` (visualization — unused on the sidecar). This bloats the sidecar environment and introduces unnecessary dependencies.

2. **No proto generation automation** — `.proto` files in `python/proto/` are manually compiled. Go-side `.pb.go` and Python-side `*_pb2.py` can drift.

3. **Python tests fail locally** — No `.venv` setup instructions. Tests require `pyarrow`, `pandas`, `grpcio` etc. which aren't installed without `pip install -e ".[dev,data]"`.

4. **Sidecar portability** (`internal/python/sidecar.go:114-147`) — `killSidecarByPort` uses `lsof -ti :<port>` which is macOS-only.

## Design

### 1. Clean up requirements.lock

**File**: `python/requirements.lock`

Regenerate from `pyproject.toml` using `pip-compile` with only declared dependencies:

```bash
cd python && pip-compile pyproject.toml --output-file=requirements.lock
```

Before regeneration, audit `pyproject.toml` dependencies:

Remove from `pyproject.toml` (or move to optional `[viz]` extras):
- `psycopg2-binary` — violates SQLite-only rule
- `fastapi`, `uvicorn` — not needed (gRPC sidecar)
- `duckdb` — not needed
- `matplotlib`, `weasyprint`, `pillow` — visualization, not sidecar's job

Add `pip-compile` target to `Taskfile.yml`:
```yaml
python:deps:
  summary: Compile Python dependencies
  dir: python
  cmds:
    - pip-compile pyproject.toml --output-file=requirements.lock
```

### 2. Automate proto generation

**File**: `Taskfile.yml`

Add proto generation target:

```yaml
proto:
  summary: Regenerate gRPC code from .proto files
  cmds:
    - cd python && python -m grpc_tools.protoc
      -I proto
      --python_out=src/proto
      --grpc_python_out=src/proto
      proto/*.proto
    - protoc --go_out=internal/python/proto --go-grpc_out=internal/python/proto
      -I python/proto
      python/proto/*.proto
    - echo "→ Proto generated. Remove _pb2.py. grpc.py files have _pb2_grpc.py analogues."
```

Document in `python/README.md` or a comment at the top of each `.proto` file:
```
# Regenerate with: task proto
```

### 3. Fix Python local dev setup

**File**: `Makefile` — Add dev setup target:

```makefile
python-setup:
	cd python && python3 -m venv .venv && \
	.venv/bin/pip install --upgrade pip && \
	.venv/bin/pip install -e ".[dev,data]"
```

**File**: `python/README.md` — Add setup instructions:
```markdown
## Local Development

```bash
cd python
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev,data]"
pytest tests/ -x -q
```
```

### 4. Fix sidecar portability (partial)

**File**: `internal/python/sidecar.go`

Replace `lsof` with a more portable approach. Options:
- **Option A** (recommended): Store the PID when starting the sidecar and use it in `Stop()`, avoiding port-based PID discovery entirely.
- **Option B**: Use `net.Listen` on the sidecar port to detect it's free, then kill using PID file.

Implement Option A:

```go
type SidecarProcess struct {
    cmd    *exec.Cmd
    pid    int
    port   int
    cancel context.CancelFunc
}

func (s *SidecarProcess) Stop() error {
    if s.cmd != nil && s.cmd.Process != nil {
        return s.cmd.Process.Signal(os.Interrupt)
    }
    return nil
}
```

Remove `killSidecarByPort` entirely. The PID is known from `cmd.Start()`.

### Modified files

| File | Change |
|------|--------|
| `python/requirements.lock` | Regenerate with cleaned dependencies |
| `python/pyproject.toml` | Remove psycopg2-binary, fastapi, uvicorn, duckdb, matplotlib |
| `Taskfile.yml` | Add `python:deps` and `proto` targets |
| `Makefile` | Add `python-setup` target |
| `python/README.md` | Add local dev setup instructions |
| `internal/python/sidecar.go` | Replace `killSidecarByPort` with PID-based stopping |
| `internal/python/sidecar_test.go` | Update tests |
| `internal/python/bridge.go` | (may need minor updates for new sidecar lifecycle) |

### API changes

- `SidecarProcess.Stop()` behavior unchanged (same interface, more reliable implementation)
- `killSidecarByPort` removed (unexported, no external callers)
- No gRPC or frontend changes

## Acceptance Criteria

- [ ] `requirements.lock` has ≤60 packages (was 198) with no Postgres/HTTP/DB bloat
- [ ] `cd python && pip-compile pyproject.toml` succeeds
- [ ] `cd python && task proto` regenerates both Go and Python proto files
- [ ] `cd python && python -m pytest tests/ -x -q` passes in fresh venv
- [ ] Sidecar `Stop()` works on macOS and Linux (no lsof dependency)
- [ ] All Go tests pass (including Python bridge tests)

## Risks / Trade-offs

- **Dependency cleanup**: Removing `matplotlib` may break existing Python code that imports it. Audit `python/src/` first for any `import matplotlib` before removing.
- **Proto generation**: Both `protoc` (Go) and `grpc_tools.protoc` (Python) must be installed. Document as dev prerequisites.
- **PID-based sidecar stop**: If the sidecar process spawns child processes, `Process.Signal` won't reach them. Mitigation: use process group (`Setpgid: true` on `exec.Cmd`).
