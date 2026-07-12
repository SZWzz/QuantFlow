"""Pytest configuration -- add src/ to sys.path for imports."""
import sys
from pathlib import Path

# Add python/src to the import path so tests can import from src.*
_src = Path(__file__).resolve().parent / "src"
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))
