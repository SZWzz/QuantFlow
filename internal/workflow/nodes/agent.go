package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/ai"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
	"quantflow/internal/workflow"
)

// AgentNode is a workflow node that runs an AI agent with tool access.
type AgentNode struct {
	id     string
	params map[string]any
}

// NewAgentNode creates a new AgentNode.
func NewAgentNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &AgentNode{id: id, params: params}, nil
}

func (n *AgentNode) ID() string       { return n.id }
func (n *AgentNode) NodeType() string { return "agent" }
func (n *AgentNode) Category() string { return "ai" }

func (n *AgentNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "prompt", Type: workflow.PortString, Required: true},
		{Name: "context", Type: workflow.PortSeries, Required: false},
		{Name: "constraints", Type: workflow.PortString, Required: false},
	}
}

func (n *AgentNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "result", Type: workflow.PortString, Required: false},
		{Name: "analysis", Type: workflow.PortSeries, Required: false},
		{Name: "signal", Type: workflow.PortSignal, Required: false},
	}
}

func (n *AgentNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "profile", Type: "string", Default: "general",
			Description: "Agent profile name (general, quant_analyst, trader, research_assistant)"},
		{Name: "model", Type: "string", Default: "",
			Description: "LLM model override (default: from profile)"},
		{Name: "max_steps", Type: "int", Default: 5,
			Description: "Maximum ReAct loop steps"},
		{Name: "temperature", Type: "float", Default: 0.7,
			Description: "LLM temperature (0.0-2.0)"},
	}
}

func (n *AgentNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var agentBridge *python.PythonBridge
	var agentReg *ai.CapabilityRegistry
	var agentEmitter *ai.EventEmitter
	var agentPM workflow.ProfileMgrService
	if nctx != nil {
		if nctx.Bridge != nil {
			agentBridge, _ = nctx.Bridge.(*python.PythonBridge)
		}
		agentReg, _ = nctx.CapRegistry.(*ai.CapabilityRegistry)
		agentEmitter, _ = nctx.Emitter.(*ai.EventEmitter)
		agentPM = nctx.ProfileMgr
	}

	profileName := getStringParam(params, "profile", "general")
	model := getStringParam(params, "model", "")
	temperature := float32(getFloatParam(params, "temperature", 0.7))

	if agentPM == nil {
		return nil, fmt.Errorf("agent: profile manager not initialized")
	}
	profile, err := agentPM.Get(profileName)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}

	maxSteps := getIntParam(params, "max_steps", profile.MaxSteps)
	profile.MaxSteps = maxSteps

	if agentBridge == nil {
		return nil, fmt.Errorf("agent: PythonBridge not initialized")
	}
	if agentReg == nil {
		return nil, fmt.Errorf("agent: CapabilityRegistry not initialized")
	}

	// Build messages from inputs
	var messages []*pb.ChatMessage

	if contextData, ok := inputs["context"]; ok && contextData != nil {
		messages = append(messages, &pb.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Here is the data for analysis:\n%v", contextData),
		})
	}

	if constraints, ok := inputs["constraints"]; ok && constraints != nil {
		messages = append(messages, &pb.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Constraints: %v", constraints),
		})
	}

	prompt, _ := inputs["prompt"].(string)
	if prompt == "" {
		prompt = getStringParam(params, "prompt", "Analyze the provided data and give insights.")
	}
	messages = append(messages, &pb.ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	loop := ai.NewAgentLoop(agentBridge, agentReg, agentEmitter)
	runID := fmt.Sprintf("wf_%s", n.id)
	result, err := loop.Run(ctx, runID, messages, profile, model, temperature)
	if err != nil && err != ai.ErrMaxStepsExceeded {
		return nil, fmt.Errorf("agent: %w", err)
	}

	outputs := map[string]any{
		"result":   result.FinalContent,
		"analysis": nil,
		"signal":   nil,
	}

	if result.FinalContent != "" {
		outputs["analysis"] = map[string]any{
			"steps":      result.Steps,
			"tokens":     result.TokenUsage,
			"tool_calls": len(result.ToolCalls),
		}
	}

	return outputs, nil
}

func (n *AgentNode) Validate() error {
	profileName := getStringParam(n.params, "profile", "general")
	validProfiles := map[string]bool{
		"general": true, "quant_analyst": true, "trader": true, "research_assistant": true,
	}
	if !validProfiles[profileName] {
		return fmt.Errorf("agent: invalid profile %q (valid: general, quant_analyst, trader, research_assistant)", profileName)
	}
	return nil
}
