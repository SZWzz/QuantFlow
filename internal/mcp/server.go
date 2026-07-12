package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Server runs the MCP JSON-RPC event loop over a Transport.
type Server struct {
	handler   *Handler
	transport *StdioTransport
}

// NewServer creates an MCP server.
func NewServer(handler *Handler) *Server {
	return &Server{
		handler:   handler,
		transport: NewStdioTransport(),
	}
}

// Run starts the JSON-RPC event loop. Blocks until ctx is cancelled
// or stdin returns io.EOF. Writes responses to stdout, logs to stderr.
func (s *Server) Run(ctx context.Context) error {
	s.transport.Log("MCP server starting (quantflow %s)", s.handler.version)

	for {
		select {
		case <-ctx.Done():
			s.transport.Log("shutting down: %v", ctx.Err())
			return nil
		default:
		}

		req, err := s.transport.ReadRequest()
		if err == io.EOF {
			s.transport.Log("client disconnected (EOF)")
			return nil
		}
		if err != nil {
			s.transport.Log("read error: %v", err)
			continue
		}

		// Handle the method.
		result, rpcErr := s.handler.HandleMethod(req.Method, req.Params)
		resp := &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
			Error:   rpcErr,
		}

		// Suppress response for notifications (no ID).
		if req.ID == nil {
			continue
		}

		if err := s.transport.WriteResponse(resp); err != nil {
			s.transport.Log("write error: %v", err)
			return fmt.Errorf("write response: %w", err)
		}
	}
}

// MustJSON is a helper for embedding JSON literals in tests.
func MustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
