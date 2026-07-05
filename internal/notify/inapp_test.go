package notify

import (
	"context"
	"testing"
)

func TestInAppNotifier_Send(t *testing.T) {
	n := NewInAppNotifier()
	msg := &Message{Title: "Test Title", Body: "Test Body", Level: LevelInfo}
	err := n.Send(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInAppNotifier_Name(t *testing.T) {
	n := NewInAppNotifier()
	if n.Name() != "inapp" {
		t.Errorf("expected 'inapp', got %q", n.Name())
	}
}
