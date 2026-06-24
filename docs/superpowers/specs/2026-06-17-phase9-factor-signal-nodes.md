# Phase 9: Factor Atoms + Signal Engineering (34 → 54)

## Design: 20 new nodes

**Factor Atoms (12)**: pct_change, delta, std_dev, rank, scale, cross_over, compare, bool_combine, rolling_maxmin, rolling_zscore, arithmetic, if_else
**Signal (5)**: rank_select, hold_signal, rebalance, entry_signal, exit_signal
**Control/Output (3)**: if_condition, sub_workflow, chart_data

## Files: 20 new .go files + register.go update
## AC: 54 total nodes, go build + test pass
