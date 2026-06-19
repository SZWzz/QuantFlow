package nodes

import (
	"context"
	"log/slog"

	"quantflow/internal/research"
	"quantflow/internal/workflow"
)

// SatelliteNode fetches satellite-derived energy data (NASA POWER + FIRMS)
// and extracts energy anomaly trading signals. Degrades to mock data when the
// satellite service is not configured.
type SatelliteNode struct {
	id     string
	params map[string]any
}

// NewSatelliteNode creates a new satellite workflow node.
func NewSatelliteNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SatelliteNode{id: id, params: params}, nil
}

func (n *SatelliteNode) ID() string       { return n.id }
func (n *SatelliteNode) NodeType() string  { return "satellite" }
func (n *SatelliteNode) Category() string  { return "alternative_data" }

func (n *SatelliteNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "region", Type: workflow.PortString, Required: false},
	}
}

func (n *SatelliteNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "energy_signal", Type: workflow.PortSignal, Required: false},
		{Name: "solar_ghi", Type: workflow.PortNumber, Required: false},
		{Name: "wind_speed", Type: workflow.PortNumber, Required: false},
	}
}

func (n *SatelliteNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "region", Type: "string", Default: "", Description: "Region ID (texas, north-sea, gobi, sahara, midwest)"},
		{Name: "anomaly_threshold", Type: "number", Default: 20, Description: "Anomaly detection threshold %"},
	}
}

func (n *SatelliteNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	region := resolveStringParam(params, n.params, "region", "")
	if v, ok := inputs["region"].(string); ok && v != "" {
		region = v
	}

	// anomaly_threshold is passed as a param for future use; currently handled by the service.
	_ = resolveFloatParam(params, n.params, "anomaly_threshold")

	var signals []research.SatelliteSignal
	var err error

	if satelliteService != nil {
		signals, err = satelliteService.ExtractSignals(ctx)
	} else {
		slog.Warn("satellite service not set, using mock")
		mockSvc := research.NewSatelliteService(nil)
		signals, _ = mockSvc.ExtractSignals(ctx)
	}
	if err != nil {
		slog.Warn("satellite signal extraction failed", "error", err)
		mockSvc := research.NewSatelliteService(nil)
		signals, _ = mockSvc.ExtractSignals(ctx)
	}

	// Find the matching region signal or use the first one
	var matchedSignal *research.SatelliteSignal
	if region != "" {
		for i := range signals {
			if signals[i].Region == region {
				matchedSignal = &signals[i]
				break
			}
		}
	}
	if matchedSignal == nil && len(signals) > 0 {
		matchedSignal = &signals[0]
	}
	if matchedSignal == nil {
		matchedSignal = &research.SatelliteSignal{
			Region:      region,
			Signal:      "neutral",
			Description: "no satellite data available",
			Confidence:  0.0,
		}
	}

	// Get the latest snapshot values for the region
	solarGHI := 0.0
	windSpeed := 0.0

	// Fetch region-specific snapshot values
	if satelliteService != nil && region != "" {
		snaps, err := satelliteService.GetRegionSnapshots(ctx)
		if err == nil {
			for _, snap := range snaps {
				if snap.ID == region {
					solarGHI = snap.SolarGHI
					windSpeed = snap.WindSpeed
					break
				}
			}
		}
	} else {
		// Mock values from the regions
		mockSvc := research.NewSatelliteService(nil)
		snaps, _ := mockSvc.GetRegionSnapshots(ctx)
		for _, snap := range snaps {
			if snap.ID == region {
				solarGHI = snap.SolarGHI
				windSpeed = snap.WindSpeed
				break
			}
		}
	}

	// Build energy signal output
	action := matchedSignal.Signal // bullish, bearish, neutral
	confidence := matchedSignal.Confidence
	description := matchedSignal.Description

	return map[string]any{
		"energy_signal": map[string]any{
			"action":      action,
			"confidence":  confidence,
			"description": description,
			"region":      matchedSignal.Region,
		},
		"solar_ghi":  solarGHI,
		"wind_speed": windSpeed,
	}, nil
}

func (n *SatelliteNode) Validate() error { return nil }
