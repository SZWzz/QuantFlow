# Python Subprocess → Direct Import Refactoring

**Phase 2 of the Python subprocess elimination plan.**

## Motivation

Every fincept module call via subprocess (`python -m src.data.fincept.xxx endpoint arg...`) pays ~200ms process-spawn overhead per request, loses structured error context (exit code + stderr only), bypasses asyncio, and makes stack traces useless. With 19+ AKShare data types and 3 macro sources routing through subprocess, users experience cumulative latency whenever multiple data types are queried (e.g., loading a stock dashboard fetches financials + fundflow + margin simultaneously → 3 subprocess spawns = ~600ms wasted).

The `crypto_extras` handler already demonstrates the correct pattern: `importlib.util.spec_from_file_location()` + in-process function calls.

## Design

### Data flow (before)

```
FetchData("akshare", "financials", symbol)
  → DataService._handle_akshare()
    → subprocess.run([sys.executable, "-m", "src.data.fincept.financials", "stock_financial_analysis_indicator_em", symbol])
      → [new process] financials.main() → akshare API → JSON stdout
    ← parse JSON from stdout
  ← return protobuf
```

### Data flow (after)

```
FetchData("akshare", "financials", symbol)
  → DataService._handle_akshare()
    → importlib.import_module("src.data.fincept.financials")
    → financials.ENDPOINTS["stock_financial_analysis_indicator_em"]["func"](symbol)
    ← native Python dict/list
  ← return protobuf
```

### Files to modify

| File | Change |
|------|--------|
| `python/src/data/fetcher.py` | Replace subprocess calls in `_handle_akshare()` and `_handle_macro()` with `importlib.import_module()` |
| `python/src/data/fincept/macro_bis.py` | Expose public async function for direct import (not just CLI `main()`) |
| `python/src/data/fincept/macro_wto.py` | Same: expose public async function |
| `python/src/data/utils.py` | Remove `_fetch_akshare_1m_subprocess` (dead code once `_handle_akshare` in fetcher.py is the only entry point) |

### Approach

#### Core technique: `importlib` dynamic dispatch

```python
import importlib

def _call_fincept_module(module_path: str, endpoint: str, *args) -> Any:
    """Import a fincept module and call its endpoint function directly. No subprocess."""
    mod = importlib.import_module(module_path)

    # Pattern A: module has ENDPOINTS dict
    endpoints = getattr(mod, "ENDPOINTS", None)
    if endpoints and endpoint in endpoints:
        func = endpoints[endpoint]["func"]
        return func(*args)

    # Pattern B: module has a Wrapper class with get_{endpoint} method
    wrapper_cls = None
    for attr_name in dir(mod):
        attr = getattr(mod, attr_name)
        if isinstance(attr, type) and attr_name.endswith("Wrapper"):
            wrapper_cls = attr
            break

    if wrapper_cls is not None:
        wrapper = wrapper_cls()
        method_name = f"get_{endpoint}" if not endpoint.startswith("get_") else endpoint
        method = getattr(wrapper, method_name, None)
        if method is not None:
            return method(*args)

    raise ValueError(f"Endpoint '{endpoint}' not found in module {module_path}")
```

#### Overall transformation strategy (in order of implementation)

The fincept modules fall into two categories based on their I/O pattern:

1. **Synchronous fincept modules** (financials, fundflow, margin, index, bonds, funds, company_info, derivatives, hk, macro_cn, macro_eia, chanlun, indicators, scanner)

   All AKShare calls are blocking I/O. Already called via `run_in_executor` in the current handling. Direct import removes the subprocess spawn but keeps the `run_in_executor` wrapping.

2. **Async fincept modules** (macro_bis, macro_wto)

   Use `aiohttp`. Currently called via subprocess which forces synchronous wait. Direct import lets us `await` natively in the asyncio event loop.

### Migration order (synchronous first)

**Step 1: Refactor `DataService._handle_akshare()`** (the 19-data-type router at line 859)

The comment on the method says "Route AKShare data types to fincept CLI modules via subprocess." Fix it to route via direct import.

The `_AKSHARE_ROUTES` dict maps `data_type → (module_name, default_endpoint)`. After step 1, this dict drives `importlib.import_module(f"src.data.fincept.{module_name}")` instead of `subprocess.run([python, "-m", f"src.data.fincept.{module_name}", endpoint, ...])`.

**Step 2: Refactor `DataService._handle_macro()`** (BIS/WTO/EIA)

Same pattern as step 1, but these are async modules. The `_handle_macro` is already an `async def` method, so we can `await` the module's async functions directly instead of wrapping them in `run_in_executor`.

**Step 3: Add gRPC endpoints for chanlun/indicators/scanner** (Go-side bridge.go elimination)

Currently Go code calls `exec.Command("python3", "-m", "src.data.fincept.chanlun", ...)` directly. Replace with gRPC calls through the existing sidecar connection. Add new RPCs to the protobuf service definition.

**Step 4: Remove dead subprocess fallback code**

- Remove `_run_akshare_subprocess()` standalone function (line 690) — never reached after step 1
- Remove `_fetch_akshare_1m_subprocess()` in utils.py — never reached
- The fallback in `_handle_akshare` standalone (line 727-733) is already correctly a fallback and can stay

## Acceptance Criteria

- [ ] All 19 AKShare data types route via direct import, verified by running each data type through the gRPC service
- [ ] Macro BIS/WTO/EIA route via direct import with proper async/await
- [ ] `subprocess.run` count in python/ drops from 4 to 0 (excluding `subprocess.Popen` for the gRPC server itself)
- [ ] All existing Python tests pass (`python -m pytest tests/ -x -q`)
- [ ] No regression in data shape (the direct import returns the same JSON structure as the subprocess)
- [ ] Chanlun/indicators/scanner accessible via gRPC

## Risks / Trade-offs

### Risk 1: Import side effects

Some fincept modules may have import-time side effects (e.g., module-level `akshare.xxx()` calls). Quick grep:
```
python/src/data/fincept$ grep -rn "^ak\." *.py | head -5
```
Only `macro_cn.py` has module-level imports (`import akshare as ak`), no side effects. Low risk.

### Risk 2: `main()` wraps errors differently

The subprocess `main()` in each module catches broad exceptions, prints error JSON, and exits:
```python
try:
    result = func(*args)
    print(json.dumps({"data": result, "success": True}, ...))
except Exception as e:
    print(json.dumps({"success": False, "error": str(e)}, ...))
```

Direct import bypasses this wrapper. We must replicate the error handling at the call site in `fetcher.py`, or add a public wrapper function to each module.

Approach: **Add a public `call_endpoint(endpoint, *args)` function** to each fincept module that wraps the error handling. This way the module-level error handling is preserved regardless of entry point (CLI or direct import).

### Risk 3: Module discovery at runtime

`importlib.import_module("src.data.fincept.financials")` requires the package to be installed in the same Python environment. Since the gRPC sidecar is launched with the project's Python (`sys.executable`) and the modules live in `src/data/fincept/` which is on `PYTHONPATH`, this should work identically to `python -m src.data.fincept.financials`.

### Risk 4: Async module incompatibility

`macro_bis.py` uses `async def main(): ...` with `asyncio.run()`. When imported directly, the functions are `async` and must be awaited. `_handle_macro()` is already an `async def`, so this is straightforward — just `await` instead of `run_in_executor`.

## Implementation plan

### Detailed tasks

#### Task 1: Add `call_endpoint()` public function to every synchronous fincept module

For each module in `_AKSHARE_ROUTES` that uses pattern A (ENDPOINTS dict), add:

```python
# At module level
def call_endpoint(endpoint: str, *args) -> Any:
    """Public entry point for direct import callers. Wraps error handling."""
    endpoints = globals().get("ENDPOINTS", {})
    if endpoint == "get_all_endpoints":
        return get_all_endpoints()
    entry = endpoints.get(endpoint)
    if entry is None:
        return {"success": False, "error": f"Unknown endpoint: {endpoint}"}
    try:
        func = entry["func"]
        sig = inspect.signature(func)
        bound = sig.bind(*args)
        result = func(*bound.args)
        return result
    except Exception as e:
        return {"success": False, "error": str(e)}
```

For pattern B (Wrapper class), add:

```python
def call_endpoint(endpoint: str, *args) -> Any:
    wrapper = WrapperClass()
    method_name = f"get_{endpoint}" if not endpoint.startswith("get_") else endpoint
    method = getattr(wrapper, method_name, None)
    if method is None:
        return {"success": False, "error": f"Unknown endpoint: {endpoint}"}
    try:
        return method(*args)
    except Exception as e:
        return {"success": False, "error": str(e)}
```

Modules to update: financials, company_info, fundflow, margin, index, bonds, funds, company_info, derivatives, hk, macro_cn, crypto_extras

For macro_cn, also add async support since `get_summary()` and `get_normalized()` use async internally:

```python
async def call_endpoint_async(endpoint: str, *args) -> Any:
    wrapper = MacroCNWrapper()
    ...
    return await method(*args)
```

#### Task 2: Refactor `DataService._handle_akshare()` to use `call_endpoint()`

Replace this (lines 859-919):
```python
async def _handle_akshare(self, data_type, symbols, start_date, end_date, params):
    import subprocess
    import sys as _sys
    ...
    cmd = [_sys.executable, "-m", f"src.data.fincept.{module_name}", endpoint]
    ...
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    ...
```

With:
```python
async def _handle_akshare(self, data_type, symbols, start_date, end_date, params):
    route = self._AKSHARE_ROUTES.get(data_type)
    module_path = f"src.data.fincept.{route[0]}"
    endpoint = route[1]
    if params and params.get("cmd"):
        endpoint = params["cmd"]
    
    mod = importlib.import_module(module_path)
    call_endpoint_fn = getattr(mod, "call_endpoint", None)
    
    # Build positional args matching the CLI contract
    symbol = symbols[0] if symbols else ""
    args = []
    if symbol and data_type not in NO_SYMBOL_TYPES:
        args.append(symbol)
    if start_date and data_type == "fundflow":
        args.append(start_date)
    
    loop = asyncio.get_event_loop()
    result = await loop.run_in_executor(None, lambda: call_endpoint_fn(endpoint, *args))
    
    if isinstance(result, dict) and not result.get("success", True):
        return data_pb2.FetchDataResponse(error=result.get("error", "unknown error"))
    
    return data_pb2.FetchDataResponse(
        data=json.dumps(result, default=str, ensure_ascii=False).encode("utf-8")
    )
```

#### Task 3: Refactor `DataService._handle_macro()` (BIS/WTO/EIA)

Same pattern but with `await` directly on `call_endpoint_async`:

```python
async def _handle_macro(self, data_type, symbols, start_date, end_date, params):
    mod = importlib.import_module(f"src.data.fincept.macro_{data_type}")
    endpoint = (params or {}).get("cmd", "get_all_endpoints")
    
    if hasattr(mod, "call_endpoint_async"):
        result = await mod.call_endpoint_async(endpoint, *cmd_args)
    elif hasattr(mod, "call_endpoint"):
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, lambda: mod.call_endpoint(endpoint, *cmd_args))
    else:
        return data_pb2.FetchDataResponse(error=f"Module {data_type} has no call_endpoint")
    
    ...
```

#### Task 4: Remove dead fallback code

- Remove `_run_akshare_subprocess()` standalone function (lines 690-721)
- Verify `_fetch_akshare_1m_subprocess()` in utils.py is unreachable and remove it

#### Task 5: Add gRPC endpoints for chanlun/indicators/scanner (Go bridge)

Add to `python/src/server.py`:
```python
class AnalysisService(analysis_pb2_grpc.AnalysisServiceServicer):
    async def RunChanlun(self, request, context):
        from src.data.fincept.chanlun import call_endpoint
        ...
    async def ComputeIndicator(self, request, context):
        ...
    async def ScanStocks(self, request, context):
        ...
```

Update Go code in `internal/python/bridge.go` to use gRPC instead of `exec.Command`.
