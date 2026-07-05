package topics

import (
	"encoding/json"
	"testing"
)

func TestTick_Marshal(t *testing.T) {
	data := Tick{
		Symbol: "AAPL",
		Price:  150.25,
		Volume: 1000,
		Time:   1700000000,
		Side:   "buy",
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
	if decoded["side"] != "buy" {
		t.Errorf("side = %v", decoded["side"])
	}
}

func TestTick_RoundTrip(t *testing.T) {
	orig := Tick{Symbol: "TSLA", Price: 250.0, Volume: 500, Time: 1700000002, Side: "sell"}
	b, _ := json.Marshal(orig)
	var got Tick
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != orig {
		t.Errorf("round-trip mismatch: %+v vs %+v", got, orig)
	}
}

func TestTick_Empty(t *testing.T) {
	data := Tick{}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}
