// Command quantflow-mcp is a standalone MCP (Model Context Protocol) server
// that exposes QuantFlow capabilities and workflow nodes as LLM-callable tools.
//
// Transport: stdio JSON-RPC (connect from Claude Desktop, Cursor, Continue, etc.)
//
// Build: go build -o bin/quantflow-mcp ./cmd/mcp
package main

import (
	"context"
	"os"
	"os/signal"
	"quantflow/internal/ai"
	"quantflow/internal/mcp"
	"quantflow/internal/workflow"
	"syscall"
)

func main() {
	// Initialize core registries (lightweight — no Wails/GUI deps).
	capReg := ai.NewCapabilityRegistry()
	nodeReg := workflow.NewRegistry()

	// Register built-in capabilities.
	// TODO: move capability registration to a shared helper (currently in app_startup.go).
	// For now, the MCP server starts with an empty registry — capabilities
	// can be populated here as needed.

	_ = nodeReg // reserved for future node→capability adapter integration

	// Build and run the MCP server.
	handler := mcp.NewHandler(capReg)
	server := mcp.NewServer(handler)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// 不使用 defer cancel()：os.Exit 会跳过 defer，两条退出路径都显式调用
	if err := server.Run(ctx); err != nil {
		cancel()
		os.Exit(1)
	}
	cancel()
}
