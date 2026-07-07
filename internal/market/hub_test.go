package market

import (
	"sync"
	"testing"
	"time"
)

func TestHub_SubscribeAndPublish(t *testing.T) {
	hub := NewHub()

	ch, unsub := hub.Subscribe("market:quote:AAPL", "test-sub")
	defer unsub()

	hub.Publish("market:quote:AAPL", &QuoteSnapshot{
		Symbol: "AAPL",
		Last:   195.32,
	})

	select {
	case msg := <-ch:
		quote, ok := msg.Data.(*QuoteSnapshot)
		if !ok {
			t.Fatalf("expected *QuoteSnapshot, got %T", msg.Data)
		}
		if quote.Last != 195.32 {
			t.Errorf("Last = %f, want 195.32", quote.Last)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestHub_MultipleSubscribers(t *testing.T) {
	hub := NewHub()

	ch1, unsub1 := hub.Subscribe("market:quote:MSFT", "sub1")
	defer unsub1()
	ch2, unsub2 := hub.Subscribe("market:quote:MSFT", "sub2")
	defer unsub2()

	hub.Publish("market:quote:MSFT", &QuoteSnapshot{Last: 378.91})

	// Both should receive
	for i, ch := range []<-chan MarketMessage{ch1, ch2} {
		select {
		case msg := <-ch:
			quote := msg.Data.(*QuoteSnapshot)
			if quote.Last != 378.91 {
				t.Errorf("subscriber %d: Last = %f", i+1, quote.Last)
			}
		case <-time.After(time.Second):
			t.Errorf("subscriber %d: timeout", i+1)
		}
	}
}

func TestHub_Unsubscribe(t *testing.T) {
	hub := NewHub()

	ch, unsub := hub.Subscribe("market:quote:TSLA", "sub")
	unsub()

	hub.Publish("market:quote:TSLA", &QuoteSnapshot{Last: 245.30})

	select {
	case <-ch:
		t.Error("should not receive after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected: no message received
	}
}

func TestHub_GetLatest(t *testing.T) {
	hub := NewHub()

	hub.Publish("market:quote:NVDA", &QuoteSnapshot{Last: 875.28})

	latest, ok := hub.GetLatest("market:quote:NVDA")
	if !ok {
		t.Fatal("expected cached message")
	}
	quote := latest.Data.(*QuoteSnapshot)
	if quote.Last != 875.28 {
		t.Errorf("Last = %f", quote.Last)
	}
}

func TestHub_CachedMessageOnSubscribe(t *testing.T) {
	hub := NewHub()

	// Publish before subscribe
	hub.Publish("market:quote:AAPL", &QuoteSnapshot{Last: 200.0})

	// Subscribe — should receive cached message immediately
	ch, unsub := hub.Subscribe("market:quote:AAPL", "late-sub")
	defer unsub()

	select {
	case msg := <-ch:
		quote := msg.Data.(*QuoteSnapshot)
		if quote.Last != 200.0 {
			t.Errorf("Last = %f, want 200.0", quote.Last)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cached message")
	}
}

func TestHub_SubscriberCount(t *testing.T) {
	hub := NewHub()

	_, unsub1 := hub.Subscribe("topic1", "s1")
	defer unsub1()
	_, unsub2 := hub.Subscribe("topic2", "s2")
	defer unsub2()
	_, unsub3 := hub.Subscribe("topic2", "s3")
	defer unsub3()

	if hub.SubscriberCount() != 3 {
		t.Errorf("SubscriberCount = %d, want 3", hub.SubscriberCount())
	}
	if hub.TopicCount() != 2 {
		t.Errorf("TopicCount = %d, want 2", hub.TopicCount())
	}
}

func TestConcurrentPublish(t *testing.T) {
	hub := NewHub()

	ch, unsub := hub.Subscribe("market:quote:RACE", "racer")
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			hub.Publish("market:quote:RACE", &QuoteSnapshot{
				Symbol: "RACE",
				Last:   float64(n),
			})
		}(i)
	}
	wg.Wait()

	// Drain all published messages (may be fewer than 10 if some are coalesced)
	received := 0
	for {
		select {
		case <-ch:
			received++
		default:
			goto done
		}
	}
done:
	if received == 0 {
		t.Fatal("expected at least 1 message from concurrent publish")
	}

	// Verify topic was created without panic or deadlock
	if hub.TopicCount() != 1 {
		t.Errorf("TopicCount = %d, want 1", hub.TopicCount())
	}
}
