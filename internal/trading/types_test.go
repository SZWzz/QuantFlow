package trading

import "testing"

func TestTradingMode_Valid(t *testing.T) {
	tests := []struct {
		mode TradingMode
		want bool
	}{
		{TradingModePaper, true},
		{TradingModeLive, true},
		{TradingMode(""), false},
		{TradingMode("invalid"), false},
	}
	for _, tt := range tests {
		if got := tt.mode.Valid(); got != tt.want {
			t.Errorf("TradingMode(%q).Valid() = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestTradingMode_IsLive(t *testing.T) {
	if TradingModeLive.IsLive() != true {
		t.Error("live mode should be live")
	}
	if TradingModePaper.IsLive() != false {
		t.Error("paper mode should not be live")
	}
}

func TestSafetyReport_Passed(t *testing.T) {
	r := SafetyReport{}
	if r.Passed() {
		t.Error("empty report should not pass")
	}
	r.Checks = []SafetyCheck{
		{Name: "Broker", OK: true, Blocking: true},
		{Name: "RiskRules", OK: true, Blocking: false},
	}
	if !r.Passed() {
		t.Error("all-ok report should pass")
	}
	r.Checks = append(r.Checks, SafetyCheck{Name: "APIKeys", OK: false, Blocking: true})
	if r.Passed() {
		t.Error("report with failing blocking check should not pass")
	}
}
