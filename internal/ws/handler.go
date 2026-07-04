package ws

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
// hub is the WebSocket hub managing topic-based subscriptions.
func ServeWS(w http.ResponseWriter, r *http.Request, hub *Hub) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("ws accept", "err", err)
		return
	}

	client := NewClient(hub, conn)
	hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
