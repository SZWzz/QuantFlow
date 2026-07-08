package main

import (
	"testing"
)

func TestTearOffWindows_MapManagement(t *testing.T) {
	a := &App{tearOffWindows: make(map[string]*tearOffEntry)}

	if ids := a.ListTearOffWindows(); len(ids) != 0 {
		t.Fatalf("expected empty, got %v", ids)
	}

	a.tearOffWindowsMu.Lock()
	a.tearOffWindows["id1"] = &tearOffEntry{
		PanelID: "watchlist", InstanceID: "id1", Label: "自选股", Params: "{}",
	}
	a.tearOffWindows["id2"] = &tearOffEntry{
		PanelID: "order-entry", InstanceID: "id2",
		Label: "交易", Params: `{"symbol":"000001"}`,
	}
	a.tearOffWindowsMu.Unlock()

	if ids := a.ListTearOffWindows(); len(ids) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(ids))
	}

	panelId, label, params, err := a.GetTearOffPanelInfo("id1")
	if err != nil {
		t.Fatalf("GetTearOffPanelInfo error: %v", err)
	}
	if panelId != "watchlist" {
		t.Errorf("panelId = %q, want %q", panelId, "watchlist")
	}
	if label != "自选股" {
		t.Errorf("label = %q, want %q", label, "自选股")
	}
	if params != "{}" {
		t.Errorf("params = %q, want %q", params, "{}")
	}

	if _, _, _, err = a.GetTearOffPanelInfo("nonexistent"); err == nil {
		t.Fatal("expected error for non-existent instanceId")
	}

	a.tearOffWindowsMu.Lock()
	delete(a.tearOffWindows, "id1")
	a.tearOffWindowsMu.Unlock()

	if ids := a.ListTearOffWindows(); len(ids) != 1 {
		t.Fatalf("expected 1 window after delete, got %d", len(ids))
	}
}
