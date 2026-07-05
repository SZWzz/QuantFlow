# Terminal → Workflow Linkage: Panel [⊕] → Workflow Node

## Motivation

Dual-mode linkage is currently one-directional: workflow nodes can pin outputs to terminal panels (`pinToTerminal`), but terminal panels have no way to spawn a corresponding workflow node. This means users who discover a useful analysis in terminal mode must manually recreate it as a workflow — a friction point that defeats the dual-mode promise.

## Design

### Data Flow

```
User clicks [⊕] on PanelHeader
  → Panel component emits action
  → workflowStore.addNode(nodeType, canvasCenter)
  → confirmDialog("切换到工作流模式?")
  → session.toggleMode() (if confirmed)
  → WorkflowMode canvas renders new node centered in viewport
```

### Mapping: panelId → nodeType

A shared `PANEL_TO_NODE` mapping defines which terminal panels can spawn which workflow nodes. Not every panel maps (e.g. `settings`, `welcome`, `system-monitor` have no workflow equivalent). The mapping lives in a new file `frontend/src/terminal/panelToNode.ts` for discoverability and easy maintenance.

### Injection Strategy

Rather than modifying 87 panel files individually, each panel receives the [⊕] button through a **composable** `useAddToWorkflow` that:

1. Checks if `PANEL_TO_NODE[panelId]` exists
2. Returns the `Control` object for PanelHeader (or null if no mapping)
3. The panel's `PanelHeader` `controls` array prepends this control

For panels that don't use `PanelHeader`'s `controls` prop (e.g., `StockScannerPanel` uses `#controls` slot), the same composable is used but injected into their custom controls area.

### New / Modified Files

| File | Action |
|------|--------|
| `frontend/src/terminal/panelToNode.ts` | **CREATE** — `PANEL_TO_NODE` mapping dict + helper |
| `frontend/src/terminal/composables/useAddToWorkflow.ts` | **CREATE** — composable returning the `Control` object |
| `frontend/src/terminal/panels/*.vue` | **MODIFY** — add `useAddToWorkflow()` to existing panels with PanelHeader controls. Start with top 20 high-value panels; remaining panels are tracked in a follow-up issue. |
| `frontend/src/stores/session.ts` | **MODIFY** — add `centerViewport()` action for positioning new nodes |
| `frontend/src/lib/i18n/en.ts` | **MODIFY** — move/rename `ml.add_to_workflow` to `workflow.add_to_workflow` if needed |
| `frontend/src/lib/i18n/zh.ts` | **MODIFY** — same |

### API Changes

None. This is a pure frontend feature:
- `workflowStore.addNode()` already exists
- `session.toggleMode()` already exists
- `confirmDialog()` already exists

### Priority Panels (Phase 1 — 20 panels)

Panels most likely to benefit from workflow integration, based on the data/analysis-to-automation pipeline:

1. `candlestick` → `data_loader`
2. `watchlist` → `loop`
3. `indicator` → `sma`, `macd`, `rsi`, `bollinger`, `ema`
4. `stock-scanner` → `rank_select`
5. `factor-analysis` → `factor`
6. `backtest` → `backtest`
7. `sentiment` → `sentiment`
8. `news` → `news_fetcher`
9. `fundflow` → `alpha` (generic data node)
10. `correlation` → `math_op`
11. `distribution` → `math_op`
12. `financials` → `financials`
13. `peer-comparison` → `peer_compare`
14. `analyst-estimates` → `analyst_estimates`
15. `insider-trading` → `insider_trades`
16. `prediction-market` → `prediction_market`
17. `geopolitics` → `geopolitics`
18. `satellite` → `satellite`
19. `macro` → `gov_data`
20. `risk-dashboard` → `risk_metrics`

For panels that map to multiple node types (`indicator`), show a submenu or pick the most common default.

## Acceptance Criteria

- [ ] `PANEL_TO_NODE` mapping covers 20+ priority panels
- [ ] Each mapped panel shows a [⊕] button in its PanelHeader controls area
- [ ] Clicking [⊕] calls `workflowStore.addNode()` with correct node type
- [ ] After adding node, a confirmation dialog asks "切换到工作流模式?" (or "Switch to workflow mode?")
- [ ] If confirmed, session switches to workflow mode with the new node centered in viewport
- [ ] If cancelled, node exists in workflow but user stays in terminal mode
- [ ] Panels without a mapping in `PANEL_TO_NODE` do NOT show the [⊕] button
- [ ] The existing `add_to_workflow` i18n key is now used (or replaced if scoped incorrectly)
- [ ] Existing `pinToTerminal` (workflow → terminal) is unaffected

## Risks / Trade-offs

- **Mapping completeness**: Not every panel maps cleanly to a single node type. Some panels aggregate multiple data sources (e.g. `market-overview`). For these, the [⊕] button is simply omitted — the user can manually assemble the workflow.
- **Indicator panel complexity**: The `indicator` panel supports many indicators. Phase 1 only maps the five most common (SMA, MACD, RSI, Bollinger, EMA). A future enhancement could show a submenu.
- **Positioning**: New nodes are placed at the workflow canvas center. If the viewport is zoomed/paned, the node may land off-screen. The `centerViewport()` action should account for current transform.
- **Non-goal**: This spec does NOT cover bidirectional sync (changing params in terminal ↔ workflow). That's a future enhancement.
