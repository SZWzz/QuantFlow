# Python Sidecar: Subprocess Anti-Pattern & Dependency Quality

## Motivation

Phase 12 review identified the Python sidecar as the project's biggest performance bottleneck and a significant maintenance risk:

1. **Subprocess-per-request** (`fetcher.py:685`): Every AKShare / macro data request spawns a full `subprocess.run(["python", ...])` (~200ms cold start). 9 concurrent macro endpoints → 9 processes. This is the #1 performance bottleneck in the entire system.

2. **Missing `[build-system]`** in `pyproject.toml`: `pip install -e .` fails without it. PEP 517 builds degrade to legacy behavior.

3. **`torch>=2.1` always installs full CUDA** (~2.5 GB). The sidecar does CPU-only inference.

4. **Groupby.apply → slow** (`cross_sectional.py:100`): 10-50× slower than `transform` for 5000-symbol universe.

5. **`mootdx` in dev optional deps** (wrong group — should be `data`).

6. **No mypy/pyright config**: Type hints are untested.

## Design

### 1. Replace Subprocess with Direct Imports

**Current flow:**
```
fetcher.py:_handle_akshare(endpoint)
  └─ subprocess.run(["python", "-c", "import akshare; ..."])
       └─ new Python process starts, imports pandas/numpy/akshare (~200ms)
```

**Target flow:**
```
fetcher.py:_handle_akshare(endpoint)
  └─ import akshare as ak  (already imported at module level)
  └─ ak.function_name(...)  (direct call, <1ms dispatch)
```

This requires refactoring the fetch functions to separate the "which function to call" logic from the "run in subprocess" logic. Many of these fetchers use `getattr(module, func_name)(**params)` patterns that work identically whether called directly or via subprocess.

**Implementation approach:**
- Wrap each subprocess call in a try/except that catches import errors
- Fall back to subprocess if the module is not available (graceful degradation)
- Cache the module reference after first successful import

**Modified files:**
- `python/src/data/fetcher.py` — Core change: replace `subprocess.run` paths with direct `importlib` + `getattr` calls
- `python/src/data/macro_fetcher.py` — Same pattern for macro data
- `python/src/data/__init__.py` — Ensure all sub-modules are importable
- `python/tests/test_data_fetcher.py` — Add unit tests (currently server-integration only)

### 2. Fix `pyproject.toml`

```toml
[build-system]
requires = ["setuptools>=64"]
build-backend = "setuptools.backends._legacy:_Backend"

[project.optional-dependencies]
# ...existing groups...
data = [
    "akshare>=1.14",
    "mootdx>=0.9",          # moved from dev
    "ccxt>=4.0",
    ...
]
dev = [
    # remove mootdx from here
]
```

**Modified files:**
- `python/pyproject.toml` — Add `[build-system]`, move `mootdx` to `data`

### 3. Torch → CPU-only Optional Dep

```toml
[project.optional-dependencies]
ml = [
    "torch>=2.1,<3",  # pip will select CPU wheel on non-CUDA systems
]
```

Add install docs: `pip install quantflow[ml]` installs CPU torch; for CUDA, users run `pip install torch --index-url https://download.pytorch.org/whl/cu121` separately.

**Modified files:**
- `python/pyproject.toml` — Split torch into `ml` optional group
- `python/README.md` — Document torch CPU vs CUDA install

### 4. Groupby.apply → transform

```python
# Before (slow):
result = df.groupby("symbol").apply(lambda g: g["volume"].rolling(5).mean() / g["close"])
result = result.reset_index(drop=True)

# After (fast):
df["vol_ratio"] = df.groupby("symbol")["volume"].transform(lambda v: v.rolling(5).mean())
df["vol_ratio"] = df["vol_ratio"] / df["close"]
```

**Modified files:**
- `python/src/factor/cross_sectional.py` — Replace `groupby.apply` with `transform`
- `python/src/factor/engine.py` — Same pattern for `zscore_volume_ratio_5d` and similar

### 5. Add pyproject.toml Tool Config

```toml
[tool.mypy]
strict = true
python_version = "3.12"
warn_unused_configs = true
warn_redundant_casts = true
warn_unused_ignores = true

[tool.pytest.ini_options]
testpaths = ["tests"]
```

**Modified files:**
- `python/pyproject.toml` — Add mymy/pytest config sections
- `.github/workflows/ci.yml` — Add mypy check step

## Acceptance Criteria

- [ ] Subprocess removal: `fetcher.py` no longer calls `subprocess.run` for any data endpoint
- [ ] Direct import fallback: If a module is not installed, falls back to subprocess with `slog.Warn`
- [ ] `pip install -e .` succeeds (build-system fixed)
- [ ] `pip install -e ".[ml]"` installs CPU-only torch (~120MB, not 2.5GB)
- [ ] Cross-sectional factor computation: same results, 10-50× faster
- [ ] `mootdx` is in `data` deps, not `dev`
- [ ] `mypy --strict src/` passes (or explicit ignores documented)
- [ ] All existing tests pass

## Risks / Trade-offs

- **Import side effects**: Some AKShare functions have side effects at import time (e.g., logging config). Test each endpoint after the change.
- **GIL blocking**: Direct imports block the asyncio event loop. Wrap CPU-heavy calls in `asyncio.to_thread` or `loop.run_in_executor`.
- **Version conflicts**: Direct imports mean the sidecar's Python env must have compatible versions of all data libraries. The virtualenv/conda setup must be maintained carefully.
