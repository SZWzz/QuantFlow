package brokers

import (
	"context"
	"testing"
)

func TestFutuBroker_Stub_ConnectReturnsError(t *testing.T) {
	broker := NewFutuBroker(FutuConfig{})
	if err := broker.Connect(context.Background()); err == nil {
		t.Error("expected error from stub Connect, got nil")
	}
}

func TestFutuBroker_Name(t *testing.T) {
	broker := NewFutuBroker(FutuConfig{})
	if broker.Name() != "futu" {
		t.Errorf("Name() = %q, want futu", broker.Name())
	}
}

func TestFutuBroker_Defaults(t *testing.T) {
	broker := NewFutuBroker(FutuConfig{})
	if broker.cfg.Host != "localhost" || broker.cfg.Port != 11111 {
		t.Errorf("defaults: host=%s port=%d, want localhost:11111", broker.cfg.Host, broker.cfg.Port)
	}
}
