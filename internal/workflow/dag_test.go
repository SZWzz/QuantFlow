package workflow

import (
	"encoding/json"
	"testing"
)

func TestWorkflow_ParseJSON(t *testing.T) {
	input := `{
		"id": "wf1",
		"name": "test workflow",
		"nodes": [
			{"id": "n1", "node_type": "data_loader", "params": {"path": "data.csv"}},
			{"id": "n2", "node_type": "sma", "params": {"period": 10}}
		],
		"edges": [
			{"from_node": "n1", "from_port": "ohlcv", "to_node": "n2", "to_port": "input"}
		]
	}`
	var wf Workflow
	if err := json.Unmarshal([]byte(input), &wf); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if wf.ID != "wf1" {
		t.Errorf("ID = %q, want wf1", wf.ID)
	}
	if len(wf.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(wf.Nodes))
	}
	if wf.Nodes[0].NodeType != "data_loader" {
		t.Errorf("Nodes[0].NodeType = %q", wf.Nodes[0].NodeType)
	}
	if len(wf.Edges) != 1 {
		t.Fatalf("len(Edges) = %d, want 1", len(wf.Edges))
	}
}

func TestWorkflow_RoundTripJSON(t *testing.T) {
	wf := Workflow{
		ID:    "wf1",
		Name:  "test",
		Nodes: []NodeInstance{{ID: "n1", NodeType: "data_loader", Params: map[string]any{"path": "data.csv"}}},
		Edges: []Edge{{FromNode: "n1", FromPort: "ohlcv", ToNode: "n2", ToPort: "input"}},
	}
	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	var wf2 Workflow
	if err := json.Unmarshal(data, &wf2); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if wf2.ID != wf.ID || wf2.Name != wf.Name {
		t.Error("round-trip mismatch")
	}
}

func TestTopoSort_SimplePipeline(t *testing.T) {
	wf := &Workflow{
		ID: "pipeline",
		Nodes: []NodeInstance{
			{ID: "a", NodeType: "passthrough"},
			{ID: "b", NodeType: "passthrough"},
			{ID: "c", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "a", FromPort: "out", ToNode: "b", ToPort: "in"},
			{FromNode: "b", FromPort: "out", ToNode: "c", ToPort: "in"},
		},
	}
	layers, err := TopoSort(wf)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	if len(layers) != 3 {
		t.Errorf("len(layers) = %d, want 3", len(layers))
	}
}

func TestTopoSort_Parallel(t *testing.T) {
	wf := &Workflow{
		ID: "fan-out",
		Nodes: []NodeInstance{
			{ID: "src", NodeType: "passthrough"},
			{ID: "a", NodeType: "passthrough"},
			{ID: "b", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "src", FromPort: "out", ToNode: "a", ToPort: "in"},
			{FromNode: "src", FromPort: "out", ToNode: "b", ToPort: "in"},
		},
	}
	layers, err := TopoSort(wf)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	if len(layers) != 2 {
		t.Errorf("len(layers) = %d, want 2", len(layers))
	}
	if len(layers[1]) != 2 {
		t.Errorf("layer2 len = %d, want 2 parallel nodes", len(layers[1]))
	}
}

func TestTopoSort_Cycle(t *testing.T) {
	wf := &Workflow{
		ID: "cycle",
		Nodes: []NodeInstance{
			{ID: "a", NodeType: "passthrough"},
			{ID: "b", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "a", FromPort: "out", ToNode: "b", ToPort: "in"},
			{FromNode: "b", FromPort: "out", ToNode: "a", ToPort: "in"},
		},
	}
	_, err := TopoSort(wf)
	if err == nil {
		t.Error("expected cycle error")
	}
}

func TestValidate_EmptyID(t *testing.T) {
	wf := &Workflow{ID: ""}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestValidate_NoNodes(t *testing.T) {
	wf := &Workflow{ID: "test"}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for no nodes")
	}
}

func TestValidate_DuplicateNodeID(t *testing.T) {
	wf := &Workflow{ID: "test", Nodes: []NodeInstance{
		{ID: "dup", NodeType: "sma"}, {ID: "dup", NodeType: "sma"},
	}}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for duplicate node ID")
	}
}

func TestValidate_UnknownEdgeEndpoint(t *testing.T) {
	wf := &Workflow{
		ID:    "test",
		Nodes: []NodeInstance{{ID: "a", NodeType: "sma"}},
		Edges: []Edge{{FromNode: "a", FromPort: "out", ToNode: "ghost", ToPort: "in"}},
	}
	err := Validate(wf)
	if err == nil {
		t.Error("expected error for unknown edge endpoint")
	}
}
