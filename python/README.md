# QuantFlow Python Sidecar

gRPC sidecar for ML, factor computation, and LLM inference.

## Local Development

```bash
cd python
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev,data]"
pytest tests/ -x -q
```

## Regenerate proto

```bash
task proto
```
