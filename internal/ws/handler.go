package ws

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

var DefaultHub = NewHub()

func init() {
	go DefaultHub.Run()
}

func ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("ws accept", "err", err)
		return
	}

	client := NewClient(DefaultHub, conn)
	DefaultHub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
