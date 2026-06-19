"""Factor evaluation: IC, IR, quantile Sharpe for a single factor formula."""
import ast
import operator
import numpy as np
import pandas as pd
import pyarrow as pa

# Allowed AST node types for safe expression evaluation.
# Only arithmetic, comparison, and a small whitelist of functions are permitted.
_ALLOWED_NODES = {
    ast.Expression, ast.BinOp, ast.UnaryOp, ast.Compare, ast.BoolOp,
    ast.Add, ast.Sub, ast.Mult, ast.Div, ast.Pow, ast.Mod,
    ast.Eq, ast.NotEq, ast.Lt, ast.LtE, ast.Gt, ast.GtE,
    ast.And, ast.Or,
    ast.USub, ast.UAdd, ast.Not,
    ast.Name, ast.Load, ast.Constant, ast.Num,  # ast.Num for Python < 3.8 compat
    ast.Call, ast.keyword,
}

_ALLOWED_FUNCTIONS = {"rank", "abs", "log", "sqrt", "sign", "max", "min"}

_OP_MAP = {
    ast.Add: operator.add, ast.Sub: operator.sub,
    ast.Mult: operator.mul, ast.Div: operator.truediv,
    ast.Pow: operator.pow, ast.Mod: operator.mod,
    ast.USub: operator.neg, ast.UAdd: operator.pos,
    ast.Eq: operator.eq, ast.NotEq: operator.ne,
    ast.Lt: operator.lt, ast.LtE: operator.le,
    ast.Gt: operator.gt, ast.GtE: operator.ge,
}

_BUILTIN_SAFE = {
    "abs": np.abs, "log": np.log, "sqrt": np.sqrt, "sign": np.sign,
    "max": np.maximum, "min": np.minimum,
}


def _safe_eval(node, namespace):
    """Recursively evaluate a whitelisted AST node against a namespace of arrays."""
    if isinstance(node, ast.Expression):
        return _safe_eval(node.body, namespace)

    if isinstance(node, ast.BinOp):
        left = _safe_eval(node.left, namespace)
        right = _safe_eval(node.right, namespace)
        op = _OP_MAP.get(type(node.op))
        if op is None:
            raise ValueError(f"unsupported operator: {type(node.op).__name__}")
        return op(left, right)

    if isinstance(node, ast.UnaryOp):
        operand = _safe_eval(node.operand, namespace)
        op = _OP_MAP.get(type(node.op))
        if op is None:
            raise ValueError(f"unsupported unary op: {type(node.op).__name__}")
        return op(operand)

    if isinstance(node, ast.Compare):
        left = _safe_eval(node.left, namespace)
        for op_node, comp in zip(node.ops, node.comparators):
            right = _safe_eval(comp, namespace)
            op = _OP_MAP.get(type(op_node))
            if op is None:
                raise ValueError(f"unsupported comparison: {type(op_node).__name__}")
            left = op(left, right)
        return left

    if isinstance(node, ast.BoolOp):
        values = [_safe_eval(v, namespace) for v in node.values]
        if isinstance(node.op, ast.And):
            result = values[0]
            for v in values[1:]:
                result = result & v
            return result
        elif isinstance(node.op, ast.Or):
            result = values[0]
            for v in values[1:]:
                result = result | v
            return result

    if isinstance(node, ast.Call):
        if isinstance(node.func, ast.Name) and node.func.id in _ALLOWED_FUNCTIONS:
            args = [_safe_eval(a, namespace) for a in node.args]
            if node.func.id == "rank":
                return pd.Series(args[0]).rank().values
            fn = _BUILTIN_SAFE[node.func.id]
            return fn(*args)
        raise ValueError(f"function call not allowed: {ast.dump(node.func)}")

    if isinstance(node, ast.Name):
        if node.id in namespace:
            return namespace[node.id]
        raise ValueError(f"unknown variable: {node.id}")

    if isinstance(node, (ast.Constant,)):  # ast.Num, ast.Str are deprecated
        return node.value

    raise ValueError(f"unsupported expression: {ast.dump(node)}")


def _validate_ast(tree):
    """Raise ValueError if the AST contains any disallowed node types."""
    for node in ast.walk(tree):
        if type(node) not in _ALLOWED_NODES:
            raise ValueError(
                f"expression contains forbidden construct: {type(node).__name__}"
            )


def evaluate_factor(formula: str, factor_data: pa.Table, returns_data: pa.Table) -> dict:
    """Evaluate a single factor formula.

    Computes Information Coefficient (IC), Information Ratio (IR), and
    a quantile-based Sharpe ratio for a factor expression evaluated against
    a returns series.

    Args:
        formula: Expression referencing factor column names (e.g., "f_0 + f_1").
                 Only arithmetic, comparison, and a small whitelist of functions
                 (rank, abs, log, sqrt, sign, max, min) are allowed.
        factor_data: Arrow Table of factor values.
        returns_data: Arrow Table with 'return' column.

    Returns:
        dict with keys: ic, ir, sharpe (and 'error' if evaluation fails).
    """
    df = factor_data.to_pandas()
    rets = returns_data.column("return").to_numpy().astype(np.float64)

    # Build evaluation namespace -- each column as a numpy array
    namespace = {col: df[col].values.astype(np.float64) for col in df.columns}

    try:
        tree = ast.parse(formula, mode="eval")
        _validate_ast(tree)
        values = _safe_eval(tree, namespace)
        values = np.asarray(values, dtype=np.float64)
    except Exception as e:
        return {"ic": 0.0, "ir": 0.0, "sharpe": 0.0, "error": str(e)}

    mask = ~np.isnan(values) & ~np.isnan(rets)
    vals, r = values[mask], rets[mask]

    if len(vals) < 30 or np.std(vals) == 0:
        return {"ic": 0.0, "ir": 0.0, "sharpe": 0.0}

    # IC (Pearson correlation between factor values and forward returns)
    ic = float(np.corrcoef(vals, r)[0, 1])
    if np.isnan(ic):
        ic = 0.0

    # IR (abs(IC) / std of rolling IC over non-overlapping segments)
    n_rolling = min(20, len(vals) // 5)
    rolling_ics = []
    for start in range(0, len(vals) - n_rolling, n_rolling):
        seg = slice(start, start + n_rolling)
        if np.std(vals[seg]) > 0:
            ric = np.corrcoef(vals[seg], r[seg])[0, 1]
            if not np.isnan(ric):
                rolling_ics.append(ric)
    ic_std = np.std(rolling_ics) if len(rolling_ics) > 1 else 1.0
    ir = float(abs(ic) / ic_std) if ic_std > 0 else 0.0

    # Quantile Sharpe (long-top-quintile / short-bottom-quintile spread)
    q = np.percentile(vals, [0, 20, 40, 60, 80, 100])
    top = vals >= q[-2]
    bot = vals <= q[1]
    long_ret = r[top].mean() if top.sum() > 0 else 0
    short_ret = -r[bot].mean() if bot.sum() > 0 else 0
    spread_std = np.std(r[top | bot]) if (top | bot).sum() > 1 else 0.01
    sharpe = (
        float((long_ret + short_ret) / spread_std * np.sqrt(252))
        if spread_std > 0
        else 0.0
    )

    return {"ic": round(ic, 6), "ir": round(ir, 4), "sharpe": round(sharpe, 4)}
