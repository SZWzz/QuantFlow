package notify

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type testNotifier struct {
	name    string
	lastMsg *Message
}

func (t *testNotifier) Send(ctx context.Context, msg *Message) error {
	t.lastMsg = msg
	return nil
}
func (t *testNotifier) Name() string { return t.name }

func TestManager_Register(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	mgr := NewManager(db)
	tn := &testNotifier{name: "test"}
	mgr.Register(tn)
	mgr.mu.RLock()
	count := len(mgr.notifiers)
	mgr.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 notifier, got %d", count)
	}
	mgr.Close()
}

func TestManager_AddNotification(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		level TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT,
		metadata TEXT DEFAULT '{}',
		is_read INTEGER DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now'))
	)`)

	mgr := NewManager(db)
	tn := &testNotifier{name: "test-add"}
	mgr.Register(tn)

	msg := &Message{Level: LevelInfo, Title: "Test Title", Body: "Test Body", Metadata: map[string]string{"key": "val"}}
	mgr.Send(msg)

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Verify via GetHistory
	history, err := mgr.GetHistory(10, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(history))
	}
	if history[0].Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", history[0].Title)
	}
	if history[0].Level != LevelInfo {
		t.Errorf("expected level 'info', got %q", history[0].Level)
	}

	// Verify notifier received the message
	if tn.lastMsg == nil {
		t.Fatal("notifier did not receive message")
	}
	if tn.lastMsg.Title != "Test Title" {
		t.Errorf("notifier title = %q, want 'Test Title'", tn.lastMsg.Title)
	}
	if tn.lastMsg.Level != LevelInfo {
		t.Errorf("notifier level = %q, want 'info'", tn.lastMsg.Level)
	}
	mgr.Close()
}

func TestManager_ListNotifications(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		level TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT,
		metadata TEXT DEFAULT '{}',
		is_read INTEGER DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now'))
	)`)

	mgr := NewManager(db)
	defer mgr.Close()

	// Send 5 notifications
	for i := 0; i < 5; i++ {
		mgr.Send(&Message{
			Level: LevelInfo,
			Title: fmt.Sprintf("Notification %d", i),
			Body:  fmt.Sprintf("Body %d", i),
		})
	}
	// Wait for async processing
	time.Sleep(150 * time.Millisecond)

	// Test limit
	history, err := mgr.GetHistory(2, 0)
	if err != nil {
		t.Fatalf("GetHistory(2,0): %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 notifications with limit=2, got %d", len(history))
	}

	// Test limit + offset
	historyOff, err := mgr.GetHistory(2, 2)
	if err != nil {
		t.Fatalf("GetHistory(2,2): %v", err)
	}
	if len(historyOff) != 2 {
		t.Errorf("expected 2 notifications with limit=2 offset=2, got %d", len(historyOff))
	}

	// Verify offset returned different items (different IDs)
	if len(history) > 0 && len(historyOff) > 0 && history[0].ID == historyOff[0].ID {
		t.Error("offset did not advance; same ID returned")
	}
}

func TestManager_MarkRead(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		level TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT,
		metadata TEXT DEFAULT '{}',
		is_read INTEGER DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now'))
	)`)

	mgr := NewManager(db)
	defer mgr.Close()

	mgr.Send(&Message{Level: LevelInfo, Title: "Read Test", Body: "Body"})
	time.Sleep(100 * time.Millisecond)

	// Get the notification
	history, err := mgr.GetHistory(10, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(history))
	}
	if history[0].IsRead {
		t.Error("expected notification to be unread initially")
	}

	// Mark as read
	if err := mgr.MarkRead(history[0].ID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}

	// Verify read status changed
	history2, err := mgr.GetHistory(10, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history2) != 1 {
		t.Fatalf("expected 1 notification after mark, got %d", len(history2))
	}
	if !history2[0].IsRead {
		t.Error("expected notification to be read after MarkRead")
	}
}

func TestLevelValues(t *testing.T) {
	if LevelInfo != "info" || LevelTrade != "trade" {
		t.Error("Level constants mismatch")
	}
}
