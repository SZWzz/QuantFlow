package research

import (
	"context"
	"testing"
)

func TestNorthboundService_GetMinuteFlow_NilAdapter(t *testing.T) {
	svc := NewNorthboundService(nil)
	data, err := svc.GetMinuteFlow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}

func TestNorthboundService_GetHistory_NilAdapter(t *testing.T) {
	svc := NewNorthboundService(nil)
	data, err := svc.GetHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}
