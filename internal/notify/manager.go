package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
)

type Manager struct {
	notifiers []Notifier
	db        *sql.DB
	eventCh   chan *Message
	mu        sync.RWMutex
}

func NewManager(db *sql.DB) *Manager {
	m := &Manager{db: db, eventCh: make(chan *Message, 256)}
	go m.processEvents()
	return m
}

func (m *Manager) Register(n Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = append(m.notifiers, n)
	slog.Info("notifier registered", "name", n.Name())
}

func (m *Manager) Send(msg *Message) {
	select {
	case m.eventCh <- msg:
	default:
		slog.Warn("notification channel full, dropping", "title", msg.Title)
	}
}

func (m *Manager) GetHistory(limit, offset int) ([]*Notification, error) {
	rows, err := m.db.Query(
		"SELECT id, level, title, body, metadata, is_read, created_at FROM notifications ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.Level, &n.Title, &n.Body, &n.Metadata, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (m *Manager) MarkRead(id int64) error {
	_, err := m.db.Exec("UPDATE notifications SET is_read = 1 WHERE id = ?", id)
	return err
}

func (m *Manager) MarkAllRead() error {
	_, err := m.db.Exec("UPDATE notifications SET is_read = 1 WHERE is_read = 0")
	return err
}

func (m *Manager) UnreadCount() int {
	var count int
	m.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE is_read = 0").Scan(&count)
	return count
}

func (m *Manager) processEvents() {
	for msg := range m.eventCh {
		m.mu.RLock()
		notifiers := make([]Notifier, len(m.notifiers))
		copy(notifiers, m.notifiers)
		m.mu.RUnlock()

		metadataJSON, _ := json.Marshal(msg.Metadata)
		m.db.Exec("INSERT INTO notifications (level, title, body, metadata) VALUES (?, ?, ?, ?)",
			msg.Level, msg.Title, msg.Body, string(metadataJSON))

		for _, n := range notifiers {
			go func(notifier Notifier) {
				if err := notifier.Send(context.Background(), msg); err != nil {
					slog.Error("notifier send failed", "channel", notifier.Name(), "error", err)
				}
			}(n)
		}
	}
}

func (m *Manager) Close() { close(m.eventCh) }
