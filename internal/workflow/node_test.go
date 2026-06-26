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

func TestCacheKeyDeterministic(t *testing.T) {
	inputsA := map[string]any{"fast": 5.0, "slow": 20.0}
	inputsB := map[string]any{"slow": 20.0, "fast": 5.0}
	keyA := CacheKey("test", inputsA)
	keyB := CacheKey("test", inputsB)
	if keyA != keyB {
		t.Errorf("CacheKey not deterministic: %q != %q", keyA, keyB)
	}
	if keyA == "" {
		t.Error("CacheKey returned empty string")
	}
}

func TestCacheKeyDifferentInputs(t *testing.T) {
	keyA := CacheKey("node", map[string]any{"period": 5})
	keyB := CacheKey("node", map[string]any{"period": 10})
	if keyA == keyB {
		t.Error("CacheKey should differ for different input values")
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
