package market

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMarketDataProvider_Interface(t *testing.T) {
	var _ MarketDataProvider = (*MarketDataHub)(nil)

	hub := NewHub()

	ch, unsub := hub.Subscribe("test_topic", "integration-test")
	defer unsub()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case msg := <-ch:
			if msg.Data != "hello" {
				t.Errorf("got data %v, want %q", msg.Data, "hello")
			}
		case <-time.After(time.Second):
			t.Error("timeout waiting for message")
		}
	}()

	hub.Publish("test_topic", "hello")
	wg.Wait()

	if n := hub.SubscriberCount(); n < 1 {
		t.Errorf("SubscriberCount = %d, want >= 1", n)
	}

	if n := hub.TopicCount(); n < 1 {
		t.Errorf("TopicCount = %d, want >= 1", n)
	}

	unsub()
	time.Sleep(10 * time.Millisecond)

	hub.Publish("test_topic", "world")
	select {
	case <-ch:
		t.Error("should not receive after unsubscribe")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestMarketDataProvider_GetLatest(t *testing.T) {
	hub := NewHub()

	if msg, ok := hub.GetLatest("nonexistent"); ok {
		t.Errorf("GetLatest nonexistent returned (%v, true), want nil, false", msg)
	}

	hub.Publish("topic1", "value1")
	time.Sleep(10 * time.Millisecond)

	msg, ok := hub.GetLatest("topic1")
	if !ok {
		t.Fatal("GetLatest after publish returned false")
	}
	if msg.Data != "value1" {
		t.Errorf("GetLatest = %v, want value1", msg.Data)
	}

	hub.Publish("topic1", "value2")
	time.Sleep(10 * time.Millisecond)

	msg, ok = hub.GetLatest("topic1")
	if !ok {
		t.Fatal("GetLatest after second publish returned false")
	}
	if msg.Data != "value2" {
		t.Errorf("GetLatest = %v, want value2", msg.Data)
	}
}

func TestMarketDataProvider_Concurrent(t *testing.T) {
	hub := NewHub()

	const n = 50
	chs := make([]<-chan MarketMessage, n)
	unsubs := make([]func(), n)

	for i := 0; i < n; i++ {
		ch, unsub := hub.Subscribe("topic_conc", fmt.Sprintf("conc-sub-%d", i))
		chs[i] = ch
		unsubs[i] = unsub
	}

	if got := hub.SubscriberCount(); got != n {
		t.Errorf("SubscriberCount = %d, want %d", got, n)
	}

	hub.Publish("topic_conc", "hello")

	for _, ch := range chs {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for concurrent subscriber")
		}
	}

	for _, unsub := range unsubs {
		unsub()
	}
}
