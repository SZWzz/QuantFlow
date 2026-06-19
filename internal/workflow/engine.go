package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

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

// Engine executes workflow DAGs layer by layer, with parallel execution
// of nodes within each topological layer. It uses an LRU cache to avoid
// re-executing nodes when the same inputs recur.
type Engine struct {
	registry *NodeRegistry
	cache    *NodeCache
}

// NewEngine creates an Engine with the given node registry and cache capacity.
func NewEngine(registry *NodeRegistry, cacheSize int) (*Engine, error) {
	cache, err := NewNodeCache(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}
	return &Engine{registry: registry, cache: cache}, nil
}

// Execute runs the workflow DAG to completion or until the context is cancelled.
// Nodes within each topological layer run in parallel via an errgroup.
// If any node or layer fails, execution stops and the partial result is returned.
func (e *Engine) Execute(ctx context.Context, wf *Workflow) (*ExecutionResult, error) {
	result := &ExecutionResult{WorkflowID: wf.ID, StartedAt: time.Now()}
	defer func() { result.FinishedAt = time.Now() }()

	if err := Validate(wf); err != nil {
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

	upstreamOutputs := &sync.Map{}
	nodeResults := make([]NodeResult, len(wf.Nodes))
	nodeResultByID := make(map[string]*NodeResult)
	for i, n := range wf.Nodes {
		nodeResults[i] = NodeResult{NodeID: n.ID, NodeType: n.NodeType}
		nodeResultByID[n.ID] = &nodeResults[i]
	}

	for layerIdx, layer := range layers {
		g, layerCtx := errgroup.WithContext(ctx)
		for _, nodeID := range layer {
			nodeID := nodeID
			layerIdx := layerIdx
			g.Go(func() error {
				return e.executeNode(layerCtx, wf, nodeID, layerIdx, upstreamOutputs, nodeResultByID)
			})
		}
		if err := g.Wait(); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.NodeResults = nodeResults
			return result, err
		}
	}

	result.Status = "success"
	result.NodeResults = nodeResults
	return result, nil
}

// executeNode instantiates and runs a single node within a workflow layer.
func (e *Engine) executeNode(ctx context.Context, wf *Workflow, nodeID string, layerIdx int, upstreamOutputs *sync.Map, nodeResultByID map[string]*NodeResult) error {
	var nodeInstance *NodeInstance
	for i := range wf.Nodes {
		if wf.Nodes[i].ID == nodeID {
			nodeInstance = &wf.Nodes[i]
			break
		}
	}
	if nodeInstance == nil {
		return fmt.Errorf("node %q not found in workflow", nodeID)
	}

	nr := nodeResultByID[nodeID]
	start := time.Now()

	node, err := e.registry.Create(nodeInstance.NodeType, nodeInstance.ID, nodeInstance.Params)
	if err != nil {
		nr.Status = "failed"
		nr.Error = err.Error()
		nr.Duration = time.Since(start)
		return fmt.Errorf("create node %q: %w", nodeID, err)
	}

	inputs := make(map[string]any)
	for _, edge := range wf.Edges {
		if edge.ToNode == nodeID {
			if val, ok := upstreamOutputs.Load(edge.FromNode); ok {
				outputs := val.(map[string]any)
				if v, ok := outputs[edge.FromPort]; ok {
					inputs[edge.ToPort] = v
				}
			}
		}
	}
	cacheKey := CacheKey(nodeID, inputs)
	if cached, ok := e.cache.Get(cacheKey); ok {
		nr.Status = "success"
		nr.Outputs = cached
		nr.Duration = time.Since(start)
		slog.Debug("cache hit", "node", nodeID, "layer", layerIdx)
		upstreamOutputs.Store(nodeID, cached)
		return nil
	}

	outputs, err := node.Execute(ctx, inputs, nodeInstance.Params)
	nr.Duration = time.Since(start)
	if err != nil {
		nr.Status = "failed"
		nr.Error = err.Error()
		return fmt.Errorf("execute node %q: %w", nodeID, err)
	}

	nr.Status = "success"
	nr.Outputs = outputs
	upstreamOutputs.Store(nodeID, outputs)
	e.cache.Put(cacheKey, outputs)
	slog.Debug("node executed", "node", nodeID, "type", nodeInstance.NodeType, "duration", nr.Duration)
	return nil
}
