// Package workflow implements a Kahn-based DAG execution engine
// for composing trading strategies from reusable node types.
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

func generateShortID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// buildUpstreamMap converts the sync.Map of upstream outputs into a regular map
// for expression resolution.
func buildUpstreamMap(upstream *sync.Map) map[string]map[string]any {
	result := make(map[string]map[string]any)
	upstream.Range(func(key, value any) bool {
		if outputs, ok := value.(map[string]any); ok {
			result[key.(string)] = outputs
		}
		return true
	})
	return result
}

// ErrorStrategy defines how the engine handles a node execution failure.
type ErrorStrategy string

const (
	ErrorStop  ErrorStrategy = "stop"  // halt the entire workflow (default)
	ErrorSkip  ErrorStrategy = "skip"  // skip this node, pass empty outputs downstream
	ErrorRetry ErrorStrategy = "retry" // retry up to N times before falling back to stop
)

// DefaultRetryCount is used when a node has retry strategy but no explicit count.
const DefaultRetryCount = 3

// retryBackoff returns the sleep duration for attempt i (0-indexed).
func retryBackoff(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
}

// ExecutionResult holds the outcome of a workflow execution, including per-node
// results, timing, and any error that caused early termination.
type ExecutionResult struct {
	WorkflowID  string       `json:"workflow_id"`
	Status      string       `json:"status"`
	StartedAt   time.Time    `json:"started_at"`
	FinishedAt  time.Time    `json:"finished_at"`
	NodeResults []NodeResult `json:"node_results"`
	Error       string       `json:"error,omitempty"`
}

// NodeResult records the execution outcome of a single node within a workflow run.
type NodeResult struct {
	NodeID   string         `json:"node_id"`
	NodeType string         `json:"node_type"`
	Status   string         `json:"status"`
	Duration time.Duration  `json:"duration"`
	Outputs  map[string]any `json:"outputs,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// ExecutionSaver is an optional callback for persisting execution results.
// When set, the engine calls it after each workflow run completes.
type ExecutionSaver func(runID string, wf *Workflow, result *ExecutionResult)

// Engine executes workflow DAGs layer by layer, with parallel execution
// of nodes within each topological layer. It uses an LRU cache to avoid
// re-executing nodes when the same inputs recur.
type Engine struct {
	registry       *NodeRegistry
	cache          *NodeCache
	nctx           *NodeContext
	executionSaver ExecutionSaver
}

// NewEngine creates an Engine with the given node registry, cache capacity, and shared
// service context accessible to all executing nodes.
func NewEngine(registry *NodeRegistry, cacheSize int, nctx *NodeContext) (*Engine, error) {
	cache, err := NewNodeCache(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}
	return &Engine{registry: registry, cache: cache, nctx: nctx}, nil
}

// SetExecutionSaver registers a callback to persist execution results after each run.
func (e *Engine) SetExecutionSaver(saver ExecutionSaver) {
	e.executionSaver = saver
}

// Execute runs the workflow DAG to completion or until the context is cancelled.
// Nodes within each topological layer run in parallel via goroutines.
// If any node or layer fails, execution stops and the partial result is returned.
func (e *Engine) Execute(ctx context.Context, wf *Workflow) (*ExecutionResult, error) {
	runID := fmt.Sprintf("run-%s-%s", time.Now().Format("20060102-150405"), generateShortID())
	result := &ExecutionResult{WorkflowID: runID, StartedAt: time.Now()}
	defer func() {
		result.FinishedAt = time.Now()
		if e.executionSaver != nil {
			e.executionSaver(runID, wf, result)
		}
	}()

	if err := Validate(wf); err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}
	if err := ValidateEdgeTypes(wf, e.registry); err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	layers, err := TopoSort(wf)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	slog.Info("executing workflow", "id", wf.ID, "name", wf.Name, "layers", len(layers), "nodes", len(wf.Nodes))

	// Inject runID into NodeContext so nodes can include it in their outputs
	if e.nctx != nil {
		e.nctx.RunID = runID
	}

	upstreamOutputs := &sync.Map{}
	nodeResults := make([]NodeResult, len(wf.Nodes))
	nodeResultByID := make(map[string]*NodeResult)
	for i, n := range wf.Nodes {
		nodeResults[i] = NodeResult{NodeID: n.ID, NodeType: n.NodeType}
		nodeResultByID[n.ID] = &nodeResults[i]
	}

	for layerIdx, layer := range layers {
		// Determine max retries for this layer (max across all nodes)
		layerMaxRetries := 0
		for _, nodeID := range layer {
			_, retryCount := getNodeErrorConfig(wf, nodeID)
			if retryCount > layerMaxRetries {
				layerMaxRetries = retryCount
			}
		}

		var layerErr error
	layerRetryLoop:
		for attempt := 0; attempt <= layerMaxRetries; attempt++ {
			if attempt > 0 {
				time.Sleep(retryBackoff(attempt - 1))
				slog.Info("retrying layer", "layer", layerIdx, "attempt", attempt+1, "max", layerMaxRetries+1)
			}

			var wg sync.WaitGroup
			var mu sync.Mutex
			layerErr = nil

			for _, nodeID := range layer {
				nodeID := nodeID
				layerIdx := layerIdx
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							mu.Lock()
							nr := nodeResultByID[nodeID]
							nr.Status = "failed"
							nr.Error = fmt.Sprintf("panic: %v", r)
							mu.Unlock()
						}
					}()
					err := e.executeNode(ctx, wf, nodeID, layerIdx, upstreamOutputs, nodeResultByID)
					if err != nil {
						mu.Lock()
						strategy, _ := getNodeErrorConfig(wf, nodeID)
						if strategy == ErrorSkip {
							upstreamOutputs.Store(nodeID, map[string]any{})
							slog.Info("node skipped due to error", "node", nodeID, "error", err)
						} else {
							layerErr = err
						}
						mu.Unlock()
					}
				}()
			}
			wg.Wait()

			if layerErr == nil {
				break layerRetryLoop
			}
		}

		if layerErr != nil {
			result.Status = "failed"
			result.Error = layerErr.Error()
			result.NodeResults = nodeResults
			return result, layerErr
		}
	}

	result.Status = "completed"
	result.NodeResults = nodeResults
	return result, nil
}

// getNodeErrorConfig extracts error strategy and retry count from a node instance.
func getNodeErrorConfig(wf *Workflow, nodeID string) (ErrorStrategy, int) {
	for i := range wf.Nodes {
		if wf.Nodes[i].ID == nodeID {
			s := ErrorStrategy(fmt.Sprint(wf.Nodes[i].Params["_onError"]))
			if s == ErrorSkip || s == ErrorRetry {
				count := 0
				if c, ok := wf.Nodes[i].Params["_retryCount"]; ok {
					switch v := c.(type) {
					case float64:
						count = int(v)
					case int:
						count = v
					}
				}
				if count <= 0 {
					count = DefaultRetryCount
				}
				return s, count
			}
			return ErrorStop, 0
		}
	}
	return ErrorStop, 0
}

// executeNode instantiates and runs a single node within a workflow layer.
func (e *Engine) executeNode(ctx context.Context, wf *Workflow, nodeID string, layerIdx int, upstreamOutputs *sync.Map, nodeResultByID map[string]*NodeResult) error {
	nr := nodeResultByID[nodeID]
	var nodeInstance *NodeInstance
	for i := range wf.Nodes {
		if wf.Nodes[i].ID == nodeID {
			nodeInstance = &wf.Nodes[i]
			break
		}
	}
	if nodeInstance == nil {
		nr.Status = "failed"
		return fmt.Errorf("node %q not found in workflow", nodeID)
	}

	start := time.Now()

	// Check pinned outputs — skip execution if this node has pinned data
	if pinned, ok := wf.PinnedOutputs[nodeID]; ok && len(pinned) > 0 {
		nr.Status = "completed"
		nr.Outputs = pinned
		nr.Duration = time.Since(start)
		upstreamOutputs.Store(nodeID, pinned)
		slog.Info("using pinned output", "node", nodeID)
		return nil
	}

	inputs := make(map[string]any)
	ancestors := make(map[string]CacheKey)
	for _, edge := range wf.Edges {
		if edge.ToNode == nodeID {
			if val, ok := upstreamOutputs.Load(edge.FromNode); ok {
				outputs := val.(map[string]any)
				if v, ok := outputs[edge.FromPort]; ok {
					inputs[edge.ToPort] = v
				}
			}
			if ck := e.cache.GetNodeKey(edge.FromNode); ck != "" {
				ancestors[edge.FromNode] = ck
			}
		}
	}
	// Resolve {{ $node.port }} expressions in params against upstream outputs
	upstreamMap := buildUpstreamMap(upstreamOutputs)
	resolvedParams, err := ResolveExpressions(nodeInstance.Params, upstreamMap, nodeID)
	if err != nil {
		nr.Status = "failed"
		nr.Error = err.Error()
		nr.Duration = time.Since(start)
		return fmt.Errorf("resolve expressions for node %q: %w", nodeID, err)
	}
	nodeInstance.Params = resolvedParams

	if cached, ok := e.cache.GetOrCompute(nodeID, nodeInstance.NodeType, resolvedParams, ancestors); ok {
		nr.Status = "completed"
		nr.Outputs = cached
		nr.Duration = time.Since(start)
		slog.Debug("cache hit", "node", nodeID, "layer", layerIdx)
		upstreamOutputs.Store(nodeID, cached)
		return nil
	}

	// Retry loop
	strategy, retryCount := getNodeErrorConfig(wf, nodeID)
	maxAttempts := 1
	if strategy == ErrorRetry {
		maxAttempts = retryCount + 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(retryBackoff(attempt - 1))
			slog.Info("retrying node", "node", nodeID, "attempt", attempt+1, "max", maxAttempts)
		}

		node, err := e.registry.Create(nodeInstance.NodeType, nodeInstance.ID, nodeInstance.Params)
		if err != nil {
			nr.Status = "failed"
			nr.Error = err.Error()
			nr.Duration = time.Since(start)
			return fmt.Errorf("create node %q: %w", nodeID, err)
		}

		outputs, err := node.Execute(ctx, inputs, nodeInstance.Params, e.nctx)
		if err == nil {
			nr.Status = "completed"
			nr.Outputs = outputs
			nr.Duration = time.Since(start)
			upstreamOutputs.Store(nodeID, outputs)
			ck := ComputeKey(nodeInstance.NodeType, resolvedParams, ancestors)
			e.cache.Store(nodeID, ck, outputs)
			slog.Debug("node executed", "node", nodeID, "type", nodeInstance.NodeType, "duration", nr.Duration)
			return nil
		}
		lastErr = err
		slog.Warn("node execution failed", "node", nodeID, "attempt", attempt+1, "error", err)
	}

	nr.Status = "failed"
	nr.Error = lastErr.Error()
	nr.Duration = time.Since(start)
	return fmt.Errorf("execute node %q: %w", nodeID, lastErr)
}
