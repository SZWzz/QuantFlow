package capabilities

import (
	"testing"

	"quantflow/internal/ai"
)

func TestRegisterFactorCapabilities_NoBridge(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterFactorCapabilities(reg, nil)

	cap := reg.GetCapability("list_factors")
	if cap == nil {
		t.Fatal("list_factors capability should be registered")
	}
	if cap.Name != "list_factors" {
		t.Errorf("name = %s, want list_factors", cap.Name)
	}
	if cap.Description == "" {
		t.Error("description should not be empty")
	}
}

func TestRegisterFactorCapabilities_ComputeFactor(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterFactorCapabilities(reg, nil)

	cap := reg.GetCapability("compute_factor")
	if cap == nil {
		t.Fatal("compute_factor capability should be registered")
	}
}

func TestListFactors_Handler_NoBridge(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterFactorCapabilities(reg, nil)

	cap := reg.GetCapability("list_factors")
	if cap == nil {
		t.Fatal("list_factors not found")
	}
	if cap.Handler == nil {
		t.Fatal("handler should not be nil")
	}
}
