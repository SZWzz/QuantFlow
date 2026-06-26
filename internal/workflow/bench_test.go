package workflow

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkEngine_100NodePipeline measures the throughput of a linear pipeline
// with 100 passthrough nodes connected in sequence.
func BenchmarkEngine_100NodePipeline(b *testing.B) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, _ := NewEngine(reg, 512, nil)

	nodes := make([]NodeInstance, 100)
	edges := make([]Edge, 0)
	for i := 0; i < 100; i++ {
		nodes[i] = NodeInstance{
			ID: fmt.Sprintf("n%d", i+1), NodeType: "passthrough",
			Params: map[string]any{"value": "data"},
		}
		if i > 0 {
			edges = append(edges, Edge{
				FromNode: fmt.Sprintf("n%d", i), FromPort: "output",
				ToNode: fmt.Sprintf("n%d", i+1), ToPort: "input",
			})
		}
	}

	wf := &Workflow{ID: "bench-100", Name: "100-node pipeline", Nodes: nodes, Edges: edges}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(context.Background(), wf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEngine_WideDAG measures the throughput of a wide fan-out/fan-in DAG
// with 1 source, 50 parallel workers, and 1 sink.
func BenchmarkEngine_WideDAG(b *testing.B) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, _ := NewEngine(reg, 512, nil)

	nodes := []NodeInstance{
		{ID: "source", NodeType: "passthrough", Params: map[string]any{"value": "data"}},
	}
	edges := make([]Edge, 0)

	for i := 0; i < 50; i++ {
		nodeID := fmt.Sprintf("worker%d", i)
		nodes = append(nodes, NodeInstance{ID: nodeID, NodeType: "passthrough"})
		edges = append(edges, Edge{FromNode: "source", FromPort: "output", ToNode: nodeID, ToPort: "input"})
	}

	nodes = append(nodes, NodeInstance{ID: "sink", NodeType: "passthrough"})
	for i := 0; i < 50; i++ {
		edges = append(edges, Edge{
			FromNode: fmt.Sprintf("worker%d", i), FromPort: "output",
			ToNode: "sink", ToPort: "input",
		})
	}

	wf := &Workflow{ID: "bench-wide", Name: "wide fan-out", Nodes: nodes, Edges: edges}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Execute(context.Background(), wf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
