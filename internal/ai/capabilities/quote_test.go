package capabilities

import (
	"testing"

	"quantflow/internal/ai"
)

func TestRegisterQuoteCapabilities_QuoteLookup(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterQuoteCapabilities(reg)

	cap := reg.GetCapability("quote_lookup")
	if cap == nil {
		t.Fatal("quote_lookup capability should be registered")
	}
	if cap.Name != "quote_lookup" {
		t.Errorf("name = %s, want quote_lookup", cap.Name)
	}
	if cap.Description == "" {
		t.Error("description should not be empty")
	}
}

func TestRegisterQuoteCapabilities_SearchSymbol(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterQuoteCapabilities(reg)

	cap := reg.GetCapability("search_symbol")
	if cap == nil {
		t.Fatal("search_symbol capability should be registered")
	}
	if cap.Name != "search_symbol" {
		t.Errorf("name = %s, want search_symbol", cap.Name)
	}
	if cap.Description == "" {
		t.Error("description should not be empty")
	}
}

func TestQuoteCapabilities_HandlerNotNil(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterQuoteCapabilities(reg)

	cap := reg.GetCapability("quote_lookup")
	if cap == nil {
		t.Fatal("quote_lookup not found")
	}
	if cap.Handler == nil {
		t.Fatal("handler should not be nil")
	}
}
