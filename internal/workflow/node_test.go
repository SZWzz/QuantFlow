package workflow

import (
	"testing"
)

func TestPortTypeConstants(t *testing.T) {
	types := map[PortType]bool{
		PortOHLCV:  true,
		PortSeries: true,
		PortSignal: true,
		PortString: true,
		PortAny:    true,
	}
	for pt, ok := range types {
		if !ok {
			t.Errorf("PortType %q should exist", pt)
		}
	}
}

func TestPortDefinition(t *testing.T) {
	pd := PortDefinition{Name: "close_price", Type: PortSeries, Required: true}
	if pd.Name != "close_price" {
		t.Errorf("Name = %q, want %q", pd.Name, "close_price")
	}
	if pd.Type != PortSeries {
		t.Errorf("Type = %q, want %q", pd.Type, PortSeries)
	}
	if !pd.Required {
		t.Error("Required should be true")
	}
}

func TestComputeKeyDeterministic(t *testing.T) {
	paramsA := map[string]any{"fast": 5.0, "slow": 20.0}
	paramsB := map[string]any{"slow": 20.0, "fast": 5.0}
	ancestors := map[string]CacheKey{"upstream": "abc123"}
	keyA := ComputeKey("test", paramsA, ancestors)
	keyB := ComputeKey("test", paramsB, ancestors)
	if keyA != keyB {
		t.Errorf("ComputeKey not deterministic: %q != %q", keyA, keyB)
	}
	if keyA == "" {
		t.Error("ComputeKey returned empty string")
	}
}

func TestComputeKeyDifferentInputs(t *testing.T) {
	keyA := ComputeKey("node", map[string]any{"period": 5}, nil)
	keyB := ComputeKey("node", map[string]any{"period": 10}, nil)
	if keyA == keyB {
		t.Error("ComputeKey should differ for different input values")
	}
}

func TestComputeKeyIncludesAncestors(t *testing.T) {
	keyNoAnc := ComputeKey("node", map[string]any{"period": 5}, nil)
	keyWithAnc := ComputeKey("node", map[string]any{"period": 5}, map[string]CacheKey{"src": "xyz"})
	if keyNoAnc == keyWithAnc {
		t.Error("ComputeKey should differ when ancestors differ")
	}
}

func TestParamDef(t *testing.T) {
	pd := ParamDef{Name: "period", Type: "int", Default: 20, Description: "SMA window"}
	if pd.Name != "period" {
		t.Errorf("Name = %q, want %q", pd.Name, "period")
	}
	if pd.Default != 20 {
		t.Errorf("Default = %v, want 20", pd.Default)
	}
}
