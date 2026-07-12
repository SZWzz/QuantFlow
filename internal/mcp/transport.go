// Package mcp implements a lightweight Model Context Protocol (MCP) server
// that exposes QuantFlow capabilities and workflow nodes as LLM-callable tools.
//
// Transport: stdio JSON-RPC (universal MCP transport — works with Claude Desktop,
// Cursor, Continue, etc.)
//
// Protocol: https://spec.modelcontextprotocol.io/specification/2024-11-05/
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StdioTransport reads JSON-RPC requests from stdin and writes responses to stdout.
// Stderr is reserved for logging (never write JSON to stderr).
type StdioTransport struct {
	reader  *bufio.Reader
	writer  io.Writer
	errLog  io.Writer
}

// NewStdioTransport creates a transport over os.Stdin / os.Stdout.
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
		errLog:  os.Stderr,
	}
}

// ReadRequest reads one JSON-RPC request from stdin. Returns io.EOF on clean shutdown.
func (t *StdioTransport) ReadRequest() (*Request, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return &req, nil
}

// WriteResponse writes one JSON-RPC response to stdout.
func (t *StdioTransport) WriteResponse(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	data = append(data, '\n')
	_, err = t.writer.Write(data)
	return err
}

// Log writes a message to stderr (not stdout — stdout is for JSON-RPC only).
func (t *StdioTransport) Log(format string, args ...any) {
	fmt.Fprintf(t.errLog, "[mcp] "+format+"\n", args...)
}
