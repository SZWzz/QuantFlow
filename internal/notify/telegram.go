package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramNotifier) Name() string { return "telegram" }

func (t *TelegramNotifier) Send(ctx context.Context, msg *Message) error {
	icon := map[Level]string{
		LevelInfo:    "ℹ️",
		LevelWarning: "⚠️",
		LevelError:   "❌",
		LevelTrade:   "\U0001f4b9",
	}[msg.Level]
	text := fmt.Sprintf("%s *%s*\n%s", icon, escapeMDV2(msg.Title), escapeMDV2(msg.Body))
	for k, v := range msg.Metadata {
		text += fmt.Sprintf("\n• %s: `%s`", escapeMDV2(k), escapeMDV2(v))
	}

	payload := map[string]interface{}{"chat_id": t.chatID, "text": text, "parse_mode": "MarkdownV2"}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	var lastErr error
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		lastErr = fmt.Errorf("telegram status %d", resp.StatusCode)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return fmt.Errorf("telegram send failed after 3 retries: %w", lastErr)
}

func escapeMDV2(s string) string {
	chars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, ch := range chars {
		s = string(bytes.ReplaceAll([]byte(s), []byte(ch), []byte("\\"+ch)))
	}
	return s
}
