package ws

import (
	"testing"
	"time"
)

func TestHubSubscribeBroadcast(t *testing.T) {
	h := NewHub()

	ch := make(chan []byte, 5)
	client := NewClient(h, nil)
	client.send = ch

	h.Subscribe(client, "test")
	h.Broadcast("test", "hello")

	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Error("empty message")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestHubUnsubscribe(t *testing.T) {
	h := NewHub()

	ch := make(chan []byte, 5)
	client := NewClient(h, nil)
	client.send = ch

	h.Subscribe(client, "test")
	h.Unsubscribe(client, "test")

	h.Broadcast("test", "data")
	select {
	case <-ch:
		t.Error("should not receive after unsubscribe")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHubBroadcastNoSubscribers(t *testing.T) {
	h := NewHub()

	h.Broadcast("nonexistent", "data")
}

func TestHubMultipleSubscribers(t *testing.T) {
	h := NewHub()

	ch1 := make(chan []byte, 5)
	ch2 := make(chan []byte, 5)
	c1 := NewClient(h, nil)
	c1.send = ch1
	c2 := NewClient(h, nil)
	c2.send = ch2

	h.Subscribe(c1, "test")
	h.Subscribe(c2, "test")
	h.Broadcast("test", "hello")

	select {
	case <-ch1:
	case <-time.After(time.Second):
		t.Error("timeout for client 1")
	}
	select {
	case <-ch2:
	case <-time.After(time.Second):
		t.Error("timeout for client 2")
	}
}

func TestHubBroadcastDifferentTopics(t *testing.T) {
	h := NewHub()

	ch := make(chan []byte, 5)
	client := NewClient(h, nil)
	client.send = ch

	h.Subscribe(client, "topic_a")
	h.Broadcast("topic_b", "data")

	select {
	case <-ch:
		t.Error("should not receive on different topic")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHubBroadcastBufferFull(t *testing.T) {
	h := NewHub()

	ch := make(chan []byte, 0)
	client := NewClient(h, nil)
	client.send = ch

	h.Subscribe(client, "test")
	h.Broadcast("test", "data")
}
