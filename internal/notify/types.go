package notify

import "context"

type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warn"
	LevelError   Level = "error"
	LevelTrade   Level = "trade"
)

type Message struct {
	Level    Level             `json:"level"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Metadata map[string]string `json:"metadata"`
}

type Notifier interface {
	Send(ctx context.Context, msg *Message) error
	Name() string
}

type Notification struct {
	ID        int64  `json:"id"`
	Level     Level  `json:"level"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Metadata  string `json:"metadata"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}
