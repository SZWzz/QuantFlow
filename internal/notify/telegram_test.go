package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramNotifier_Send_Success(t *testing.T) {
	var reqBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&reqBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := &TelegramNotifier{
		botToken: "test-key",
		chatID:   "test-chat",
		client:   server.Client(),
		baseURL:  server.URL,
	}
	msg := &Message{Title: "Test", Body: "Body", Level: LevelInfo}
	err := n.Send(context.Background(), msg)
	if err != nil {
		t.Fatal("expected no error, got:", err)
	}
	if reqBody == nil {
		t.Fatal("request body was not decoded")
	}
	if chatID, ok := reqBody["chat_id"].(string); !ok || chatID != "test-chat" {
		t.Errorf("chat_id = %v", reqBody["chat_id"])
	}
}

func TestTelegramNotifier_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	n := &TelegramNotifier{
		botToken: "test-key",
		chatID:   "test-chat",
		client:   server.Client(),
		baseURL:  server.URL,
	}
	msg := &Message{Title: "Test", Body: "Body", Level: LevelInfo}
	err := n.Send(context.Background(), msg)
	if err == nil {
		t.Error("expected error for HTTP 403")
	}
}

func TestEscapeMDV2(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"a_b", `a\_b`},
		{"a*b", `a\*b`},
		{"a[b", `a\[b`},
	}
	for _, tc := range tests {
		got := escapeMDV2(tc.input)
		if got != tc.want {
			t.Errorf("escapeMDV2(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
