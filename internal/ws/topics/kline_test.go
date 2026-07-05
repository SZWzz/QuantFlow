package topics

import (
	"encoding/json"
	"testing"
)

func TestKlineUpdate_Marshal(t *testing.T) {
	data := KlineUpdate{
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Time:     1700000000,
		Open:     50000.0,
		High:     50100.0,
		Low:      49900.0,
		Close:    50050.0,
		Volume:   100.5,
		IsClosed: true,
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal("invalid JSON:", err)
	}
	if decoded["symbol"] != "BTCUSDT" {
		t.Errorf("symbol = %v", decoded["symbol"])
	}
	if decoded["interval"] != "1m" {
		t.Errorf("interval = %v", decoded["interval"])
	}
}

func TestKlineUpdate_AllFields(t *testing.T) {
	data := KlineUpdate{
		Symbol: "ETHUSDT", Interval: "5m", Time: 1700000001,
		Open: 3000, High: 3100, Low: 2900, Close: 3050, Volume: 500, IsClosed: false,
	}
	b, _ := json.Marshal(data)
	var decoded KlineUpdate
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Open != 3000 || decoded.High != 3100 {
		t.Error("field mismatch after round-trip")
	}
}

func TestKlineUpdate_Empty(t *testing.T) {
	data := KlineUpdate{}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty JSON")
	}
}
