package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQOSMinuteAdapter_Name(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	if a.Name() != "qos" {
		t.Errorf("Name() = %s, want qos", a.Name())
	}
}

func TestQOSMinuteAdapter_Markets(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	mkts := a.Markets()
	if len(mkts) != 1 || mkts[0] != "HK" {
		t.Errorf("Markets() = %v, want [HK]", mkts)
	}
}

func TestQOSMinuteAdapter_RequiresAuth(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	if !a.RequiresAuth() {
		t.Error("expected RequiresAuth=true")
	}
}

func TestQOSMinuteAdapter_IsAvailable_NoKey(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	if a.IsAvailable(context.Background()) {
		t.Error("expected false with no API key")
	}
}

func TestQOSMinuteAdapter_IsAvailable_WithKey(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{APIKey: "test-key"})
	if !a.IsAvailable(context.Background()) {
		t.Error("expected true with API key")
	}
}

func TestQOSMinuteAdapter_FetchQuote_NotImplemented(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	_, err := a.FetchQuote(context.Background(), "00700")
	if err == nil {
		t.Error("expected error for FetchQuote")
	}
}

func TestQOSMinuteAdapter_FetchMinuteLine_NoKey(t *testing.T) {
	a := NewQOSMinuteAdapter(QOSConfig{})
	_, err := a.FetchMinuteLine("00700")
	if err == nil {
		t.Error("expected error with no API key")
	}
}

func TestQOSMinuteAdapter_FetchMinuteLine_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if req["key"] != "test-key" {
			t.Error("wrong API key")
		}
		resp := map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": []map[string]any{
				{"k": []map[string]any{
					{"t": "0930", "o": 100.0, "h": 101.0, "l": 99.5, "c": 100.5, "v": 10000},
					{"t": "0931", "o": 101.0, "h": 102.0, "l": 100.5, "c": 101.5, "v": 15000},
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	a := NewQOSMinuteAdapter(QOSConfig{APIKey: "test-key", BaseURL: server.URL})
	ticks, err := a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ticks) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(ticks))
	}
	if ticks[0].Time != "09:30" || ticks[0].Price != 100.5 || ticks[0].Volume != 10000 {
		t.Errorf("unexpected tick 0: %+v", ticks[0])
	}
}

func TestQOSMinuteAdapter_Cooldown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"code": 0, "msg": "ok", "data": []map[string]any{{"k": []map[string]any{}}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	a := NewQOSMinuteAdapter(QOSConfig{APIKey: "key", BaseURL: server.URL})

	ticks, err := a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = ticks

	ticks, err = a.FetchMinuteLine("00700")
	if err != nil {
		t.Fatalf("unexpected error on cooldown: %v", err)
	}
	if ticks != nil {
		t.Error("expected nil ticks during cooldown")
	}
}

func TestToQOSSymbol(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"00700", "HK:700"},
		{"00005", "HK:5"},
		{"700", "HK:700"},
		{"00001", "HK:1"},
		{"09988", "HK:9988"},
	}
	for _, tt := range tests {
		got := toQOSSymbol(tt.in)
		if got != tt.out {
			t.Errorf("toQOSSymbol(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
