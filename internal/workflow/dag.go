package workflow

import "fmt"

// Edge represents a directed connection between two node ports in a workflow DAG.
type Edge struct {
	FromNode string `json:"from_node"`
	FromPort string `json:"from_port"`
	ToNode   string `json:"to_node"`
	ToPort   string `json:"to_port"`
}

// NodeInstance is a concrete instantiation of a node type within a workflow,
// with its ID, type name, and runtime parameters.
type NodeInstance struct {
	ID       string         `json:"id"`
	NodeType string         `json:"node_type"`
	Params   map[string]any `json:"params,omitempty"`
}

// Workflow is the top-level model for a directed acyclic graph of node instances
// connected by edges. It is serializable to/from JSON for persistence.
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Nodes       []NodeInstance `json:"nodes"`
	Edges       []Edge         `json:"edges"`
	// PinnedOutputs maps node ID → fixed outputs for pinned/debug nodes.
	// When set, the engine skips execution of these nodes and uses the pinned data.
	PinnedOutputs map[string]map[string]any `json:"pinned_outputs,omitempty"`
}

// Clone returns a deep copy of the workflow suitable for independent execution.
func (wf *Workflow) Clone() *Workflow {
	nodes := make([]NodeInstance, len(wf.Nodes))
	for i, n := range wf.Nodes {
		nodes[i] = NodeInstance{ID: n.ID, NodeType: n.NodeType}
		if n.Params != nil {
			nodes[i].Params = make(map[string]any, len(n.Params))
			for k, v := range n.Params {
				nodes[i].Params[k] = v
			}
		}
	}
	edges := make([]Edge, len(wf.Edges))
	copy(edges, wf.Edges)
	return &Workflow{
		ID:          wf.ID + "-clone",
		Name:        wf.Name,
		Description: wf.Description,
		Nodes:       nodes,
		Edges:       edges,
	}
}

// TopoLayer is a single level in the topological ordering — all nodes in a layer
// have no dependencies on each other and can execute in parallel.
type TopoLayer []string

// TopoSort returns a topological ordering of the workflow nodes in layers
// (Kahn's algorithm). Nodes without dependencies appear in earlier layers.
// Returns a CycleError if the graph contains a cycle.
func TopoSort(wf *Workflow) ([]TopoLayer, error) {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	for _, n := range wf.Nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range wf.Edges {
		adj[e.FromNode] = append(adj[e.FromNode], e.ToNode)
		inDegree[e.ToNode]++
	}

	var queue []string
	for _, n := range wf.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var layers []TopoLayer
	for len(queue) > 0 {
		layer := make(TopoLayer, len(queue))
		copy(layer, queue)
		layers = append(layers, layer)

		var next []string
		for _, nodeID := range queue {
			for _, neighbor := range adj[nodeID] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					next = append(next, neighbor)
				}
			}
		}
		queue = next
	}

	for _, deg := range inDegree {
		if deg > 0 {
			return nil, &CycleError{Message: "workflow contains a cycle"}
		}
	}
	return layers, nil
}

// TypesCompatible returns true if an output port type can connect to an input port type.
func TypesCompatible(output, input PortType) bool {
	if output == PortAny || input == PortAny {
		return true
	}
	if output == input {
		return true
	}
	// ohlcv is compatible with series
	if output == PortOHLCV && input == PortSeries {
		return true
	}
	// signal is compatible with number
	if output == PortSignal && input == PortNumber {
		return true
	}
	return false
}

// findNode returns a pointer to the node instance with the given ID, or nil.
func findNode(nodes []NodeInstance, id string) *NodeInstance {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

// CycleError indicates a cycle was detected during topological sort.
type CycleError struct {
	Message string
}

func (e *CycleError) Error() string { return e.Message }

// ValidateEdgeTypes checks that all edges connect compatible port types.
// Requires a NodeRegistry to instantiate temporary nodes for port introspection.
func ValidateEdgeTypes(wf *Workflow, registry *NodeRegistry) error {
	if registry == nil {
		return nil // skip if no registry
	}
	for _, edge := range wf.Edges {
		srcNode := findNode(wf.Nodes, edge.FromNode)
		dstNode := findNode(wf.Nodes, edge.ToNode)
		if srcNode == nil || dstNode == nil {
			continue // structural check already done by Validate()
		}
		srcImpl, err := registry.Create(srcNode.NodeType, "_val", nil)
		if err != nil {
			continue
		}
		dstImpl, err := registry.Create(dstNode.NodeType, "_val", nil)
		if err != nil {
			continue
		}
		var srcType PortType
		for _, p := range srcImpl.OutputPorts() {
			if p.Name == edge.FromPort {
				srcType = p.Type
				break
			}
		}
		var dstType PortType
		for _, p := range dstImpl.InputPorts() {
			if p.Name == edge.ToPort {
				dstType = p.Type
				break
			}
		}
		if srcType != "" && dstType != "" && !TypesCompatible(srcType, dstType) {
			return &ValidationError{
				Message: fmt.Sprintf("edge %s.%s → %s.%s: type mismatch %s → %s",
					edge.FromNode, edge.FromPort, edge.ToNode, edge.ToPort, srcType, dstType),
			}
		}
	}
	return nil
}

// Validate checks a workflow for structural correctness:
//   - workflow ID is present
//   - at least one node exists
//   - all node IDs are non-empty and unique
//   - all node types are non-empty
//   - all edge endpoints reference existing node IDs
//   - the graph is acyclic
//
// Returns a ValidationError describing the first violation found.
func Validate(wf *Workflow) error {
	if wf.ID == "" {
		return &ValidationError{Message: "workflow id is required"}
	}
	if len(wf.Nodes) == 0 {
		return &ValidationError{Message: "workflow must have at least one node"}
	}

	nodeIDs := make(map[string]bool)
	for _, n := range wf.Nodes {
		if n.ID == "" {
			return &ValidationError{Message: "node id is required"}
		}
		if n.NodeType == "" {
			return &ValidationError{Message: "node type is required for node " + n.ID}
		}
		if nodeIDs[n.ID] {
			return &ValidationError{Message: "duplicate node id: " + n.ID}
		}
		nodeIDs[n.ID] = true
	}

	for _, e := range wf.Edges {
		if !nodeIDs[e.FromNode] {
			return &ValidationError{Message: "edge from unknown node: " + e.FromNode}
		}
		if !nodeIDs[e.ToNode] {
			return &ValidationError{Message: "edge to unknown node: " + e.ToNode}
		}
	}

	if _, err := TopoSort(wf); err != nil {
		return err
	}
	return nil
}

// ValidationError indicates a structural validation failure in a workflow.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
