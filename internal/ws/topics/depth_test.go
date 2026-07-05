package topics

import (
	"encoding/json"
	"testing"
)

func TestDepthUpdate_Marshal(t *testing.T) {
	data := DepthUpdate{
		Symbol: "AAPL",
		Bids:   []DepthLevel{{Price: 150.0, Volume: 100}},
		Asks:   []DepthLevel{{Price: 151.0, Volume: 200}},
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal("invalid JSON:", err)
	}
	if decoded["symbol"] != "AAPL" {
		t.Errorf("symbol = %v", decoded["symbol"])
	}
}

func TestDepthUpdate_Empty(t *testing.T) {
	data := DepthUpdate{Symbol: "EMPTY"}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}

func TestDepthUpdate_PriceLevel(t *testing.T) {
	level := DepthLevel{Price: 100.5, Volume: 50}
	b, err := json.Marshal(level)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if got != `{"price":100.5,"volume":50}` {
		t.Errorf("unexpected JSON: %s", got)
	}
}
