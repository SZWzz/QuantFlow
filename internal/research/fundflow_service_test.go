package research

import (
	"context"
	"testing"
)

func TestFundFlowService_GetMinuteFlow_NilAdapter(t *testing.T) {
	svc := NewFundFlowService(nil)
	data, err := svc.GetMinuteFlow(context.Background(), "000001")
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}

func TestFundFlowService_GetDailyFlow_NilAdapter(t *testing.T) {
	svc := NewFundFlowService(nil)
	data, err := svc.GetDailyFlow(context.Background(), "000001")
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}
