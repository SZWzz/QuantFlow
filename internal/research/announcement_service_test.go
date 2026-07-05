package research

import (
	"context"
	"testing"
)

func TestAnnouncementService_GetAnnouncements_NilAdapter(t *testing.T) {
	svc := NewAnnouncementService(nil)
	data, err := svc.GetAnnouncements(context.Background(), "000001", 10)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}

func TestAnnouncementService_GetAnnouncements_ZeroPageSize(t *testing.T) {
	svc := NewAnnouncementService(nil)
	data, err := svc.GetAnnouncements(context.Background(), "000001", 0)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil regardless of pageSize")
	}
}
