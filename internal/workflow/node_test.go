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

func TestParamDef(t *testing.T) {
	pd := ParamDef{Name: "period", Type: "int", Default: 20, Description: "SMA window"}
	if pd.Name != "period" {
		t.Errorf("Name = %q, want %q", pd.Name, "period")
	}
	if pd.Default != 20 {
		t.Errorf("Default = %v, want 20", pd.Default)
	}
}
