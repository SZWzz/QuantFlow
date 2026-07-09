"""Tests for fetcher.py module-level documentation and import hygiene."""

def test_fetcher_has_module_docstring():
    """Module-level docstring must exist and be non-empty."""
    from src.data import fetcher
    assert fetcher.__doc__ is not None and len(fetcher.__doc__) > 20


def test_fetcher_imports_no_identity_alias():
    """Avoid 'import X as X' pattern (redundant alias)."""
    import ast
    import inspect
    from src.data import fetcher
    source = inspect.getsource(fetcher)
    tree = ast.parse(source)
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                if alias.asname is not None:
                    assert alias.name != alias.asname, (
                        f"redundant identity alias: import {alias.name} as {alias.asname}"
                    )