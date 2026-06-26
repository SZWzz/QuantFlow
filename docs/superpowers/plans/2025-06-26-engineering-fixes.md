# QuantFlow P1 Engineering Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Implement 2 P1 engineering fixes: (1) Split monolithic `app.go` into 4 domain files, (2) Replace global `nodes.SetXxx()` setters with `NodeContext` dependency injection pattern.

**Architecture:** Fix 1 splits a ~1350-line file into 5 focused files by domain (market/trading/research/system). Fix 2 introduces a `NodeContext` struct in the `workflow` package, updates the `BaseNode` interface to accept it, and removes all package-level global variables from the `nodes` package — replacing them with explicit dependency injection through the engine.

**Tech Stack:** Go 1.22+, Wails v3

## Global Constraints

- Same `package main` for all split files — they access `a.marketReg` etc directly
- Don't refactor method logic — move ONLY, don't change
- Keep imports correct per file
- NodeContext uses `interface{}` fields to avoid import cycles
- Backward compatibility: nil check `nctx` so nodes that don't need services still work
- Verify each fix with `go build -o /dev/null .`
- Fix 2 also requires `go test ./internal/workflow/... -count=1`

---
