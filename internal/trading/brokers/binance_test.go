package brokers

import (
	"testing"
)

func TestNormalizeBinanceSymbol(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"BTC", "BTCUSDT"},
		{"ETHUSDT", "ETHUSDT"},
		{"000001.SZ", "000001SZUSDT"},
		{"AAPL", "AAPLUSDT"},
	}
	for _, tt := range tests {
		result := normalizeBinanceSymbol(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeBinanceSymbol(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBinanceStatusToOrderStatus(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"NEW", "pending"},
		{"PARTIALLY_FILLED", "partial"},
		{"FILLED", "filled"},
		{"CANCELED", "cancelled"},
		{"REJECTED", "rejected"},
	}
	for _, tt := range tests {
		result := binanceStatusToOrderStatus(tt.input)
		if string(result) != tt.expected {
			t.Errorf("binanceStatusToOrderStatus(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewBinanceBroker_Defaults(t *testing.T) {
	broker := NewBinanceBroker(BinanceConfig{APIKey: "k", SecretKey: "s"})
	if broker.Name() != "binance" {
		t.Errorf("Name() = %q, want binance", broker.Name())
	}
	if broker.IsConnected() {
		t.Error("should not be connected before Connect()")
	}
}

func TestNewBinanceBroker_Testnet(t *testing.T) {
	broker := NewBinanceBroker(BinanceConfig{APIKey: "k", SecretKey: "s", UseTestnet: true})
	if broker.cfg.BaseURL != "https://testnet.binance.vision" {
		t.Errorf("testnet BaseURL = %q", broker.cfg.BaseURL)
	}
}
