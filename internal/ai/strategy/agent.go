// Package strategy provides AI-powered trading strategy generation and iteration.
package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"quantflow/internal/workflow"
)

// LLMCaller abstracts LLM invocation.
type LLMCaller interface {
	Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// StrategyAgent generates workflow DAGs from natural language descriptions.
type StrategyAgent struct {
	llm      LLMCaller
	registry *workflow.NodeRegistry
	maxTries int
}

// NewStrategyAgent creates a strategy generation agent.
func NewStrategyAgent(llm LLMCaller, registry *workflow.NodeRegistry) *StrategyAgent {
	return &StrategyAgent{llm: llm, registry: registry, maxTries: 3}
}

// GenerateRequest is the input for strategy generation.
type GenerateRequest struct {
	Description string `json:"description"`
	Market      string `json:"market"`    // "CN" | "US" | "HK" | "CRYPTO"
	MaxNodes    int    `json:"max_nodes"` // default 15
}

// GenerateResponse is the output of strategy generation.
type GenerateResponse struct {
	Workflow *workflow.Workflow `json:"workflow"`
	RawJSON  string             `json:"raw_json"`
	Tries    int                `json:"tries"`
	Warnings []string           `json:"warnings,omitempty"`
}

// Generate creates a workflow DAG from a natural language description.
// Retries up to maxTries on validation failure, feeding errors back to the LLM.
func (a *StrategyAgent) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if req.MaxNodes <= 0 {
		req.MaxNodes = 15
	}

	systemPrompt := a.buildSystemPrompt(req.Market)
	var lastJSON string
	var validationErrors []string

	for attempt := 1; attempt <= a.maxTries; attempt++ {
		userPrompt := req.Description
		if len(validationErrors) > 0 {
			userPrompt = fmt.Sprintf(
				"Previous attempt had errors. Fix them.\n\nOriginal: %s\n\nErrors:\n%s",
				req.Description, strings.Join(validationErrors, "\n"))
		}

		response, err := a.llm.Chat(ctx, systemPrompt, userPrompt)
		if err != nil {
			return nil, fmt.Errorf("strategy agent: LLM call failed (attempt %d): %w", attempt, err)
		}

		jsonStr := extractJSON(response)
		lastJSON = jsonStr

		wf, errs := a.parseAndValidate(jsonStr, req.MaxNodes)
		if len(errs) == 0 && wf != nil {
			return &GenerateResponse{Workflow: wf, RawJSON: jsonStr, Tries: attempt}, nil
		}
		validationErrors = errs
	}

	return &GenerateResponse{RawJSON: lastJSON, Tries: a.maxTries, Warnings: validationErrors},
		fmt.Errorf("strategy agent: failed after %d attempts: %s", a.maxTries, strings.Join(validationErrors, "; "))
}

func (a *StrategyAgent) buildSystemPrompt(market string) string {
	catalog := a.buildNodeCatalog()
	return fmt.Sprintf(`You are a quantitative trading strategy expert. Generate a workflow DAG from the user's description.

## Available Nodes
%s

## Output Format (JSON only, no markdown)
{
  "name": "strategy name",
  "description": "what this does",
  "nodes": [{"id": "node_1", "type": "node_type", "params": {"key": value}}],
  "edges": [{"source": "node_1", "sourcePort": "output", "target": "node_2", "targetPort": "input"}]
}

## Rules
1. Node IDs must be unique (node_1, node_2, ...)
2. Edges must form a DAG — no cycles
3. Port types must be compatible (ohlcv→ohlcv|series|any, series→series|signal|any)
4. Start from a data source (data_loader)
5. Market: %s.`, catalog, marketContext(market))
}

func (a *StrategyAgent) buildNodeCatalog() string {
	metas := a.registry.ListAll()
	var sb strings.Builder
	catMap := make(map[string][]workflow.NodeMeta)
	for _, m := range metas {
		catMap[m.Category] = append(catMap[m.Category], m)
	}
	for cat, nodes := range catMap {
		sb.WriteString(fmt.Sprintf("### %s\n", cat))
		for _, n := range nodes {
			paramStrs := make([]string, 0)
			for _, p := range n.Params {
				paramStrs = append(paramStrs, fmt.Sprintf("%s(%s)", p.Name, p.Type))
			}
			in := portNames(n.InputPorts)
			out := portNames(n.OutputPorts)
			sb.WriteString(fmt.Sprintf("- %s [in:%s out:%s] %s\n", n.NodeType, in, out, strings.Join(paramStrs, " ")))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (a *StrategyAgent) parseAndValidate(jsonStr string, maxNodes int) (*workflow.Workflow, []string) {
	var raw struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Nodes       []struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Params map[string]any `json:"params"`
		} `json:"nodes"`
		Edges []struct {
			Source     string `json:"source"`
			SourcePort string `json:"sourcePort"`
			Target     string `json:"target"`
			TargetPort string `json:"targetPort"`
		} `json:"edges"`
	}
	var errs []string

	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, []string{fmt.Sprintf("JSON parse error: %v", err)}
	}
	if len(raw.Nodes) == 0 {
		return nil, []string{"must have at least 1 node"}
	}
	if len(raw.Nodes) > maxNodes {
		return nil, []string{fmt.Sprintf("too many nodes: %d (max %d)", len(raw.Nodes), maxNodes)}
	}

	wf := &workflow.Workflow{Name: raw.Name, Description: raw.Description}
	nodeIDs := make(map[string]bool)
	for _, n := range raw.Nodes {
		if n.ID == "" {
			errs = append(errs, "node has empty id")
			continue
		}
		if nodeIDs[n.ID] {
			errs = append(errs, fmt.Sprintf("duplicate node id: %s", n.ID))
			continue
		}
		nodeIDs[n.ID] = true

		if _, err := a.registry.Create(n.Type, "_validate", nil); err != nil {
			errs = append(errs, fmt.Sprintf("unknown node type: %s", n.Type))
			continue
		}
		wf.Nodes = append(wf.Nodes, workflow.NodeInstance{ID: n.ID, NodeType: n.Type, Params: n.Params})
	}

	for _, e := range raw.Edges {
		if !nodeIDs[e.Source] {
			errs = append(errs, fmt.Sprintf("edge source '%s' not found", e.Source))
			continue
		}
		if !nodeIDs[e.Target] {
			errs = append(errs, fmt.Sprintf("edge target '%s' not found", e.Target))
			continue
		}
		wf.Edges = append(wf.Edges, workflow.Edge{
			FromNode: e.Source, FromPort: e.SourcePort,
			ToNode: e.Target, ToPort: e.TargetPort,
		})
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Structural validation (DAG acyclic check)
	if err := workflow.Validate(wf); err != nil {
		return nil, []string{fmt.Sprintf("validation error: %v", err)}
	}

	// Type validation
	if err := workflow.ValidateEdgeTypes(wf, a.registry); err != nil {
		return nil, []string{fmt.Sprintf("type error: %v", err)}
	}

	return wf, nil
}

func portNames(ports []workflow.NodePortInfo) []string {
	names := make([]string, len(ports))
	for i, p := range ports {
		names[i] = fmt.Sprintf("%s:%s", p.Name, p.Type)
	}
	return names
}

func marketContext(market string) string {
	switch strings.ToUpper(market) {
	case "CN":
		return "A-share: T+1, price limits, min lot 100, stamp duty 0.05% sell only"
	case "US":
		return "US: PDT rule, fractional shares, no stamp duty"
	case "HK":
		return "HK: stamp duty 0.13% both sides"
	case "CRYPTO":
		return "Crypto: 24/7, no limits, funding rates"
	default:
		return "Generic market"
	}
}

func extractJSON(response string) string {
	if i := strings.Index(response, "```json"); i >= 0 {
		start := i + 7
		if end := strings.Index(response[start:], "```"); end >= 0 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	if i := strings.Index(response, "```"); i >= 0 {
		start := i + 3
		if end := strings.Index(response[start:], "```"); end >= 0 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	if i := strings.Index(response, "{"); i >= 0 {
		if end := strings.LastIndex(response, "}"); end > i {
			return response[i : end+1]
		}
	}
	return response
}
