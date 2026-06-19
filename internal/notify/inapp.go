package notify

import (
	"context"
	"log/slog"
)

type InAppNotifier struct{}

func NewInAppNotifier() *InAppNotifier { return &InAppNotifier{} }
func (n *InAppNotifier) Name() string  { return "inapp" }
func (n *InAppNotifier) Send(ctx context.Context, msg *Message) error {
	slog.Debug("inapp notification", "level", msg.Level, "title", msg.Title)
	return nil
}
