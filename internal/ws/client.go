package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/coder/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufSize    = 256
)

type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	topics  map[string]bool
	hubDone <-chan struct{}
}

type subscribeMessage struct {
	Type   string   `json:"type"`
	Topics []string `json:"topics"`
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, sendBufSize),
		topics:  make(map[string]bool),
		hubDone: hub.done,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "connection closed")
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), pongWait)
		_, msg, err := c.conn.Read(ctx)
		cancel()
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				slog.Error("ws read error", "err", err)
			}
			break
		}

		var sub subscribeMessage
		if err := json.Unmarshal(msg, &sub); err != nil {
			continue
		}

		switch sub.Type {
		case "subscribe":
			for _, topic := range sub.Topics {
				c.hub.Subscribe(c, topic)
			}
		case "unsubscribe":
			for _, topic := range sub.Topics {
				c.hub.Unsubscribe(c, topic)
			}
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close(websocket.StatusNormalClosure, "connection closed")
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				ctx, cancel := context.WithTimeout(context.Background(), writeWait)
				defer cancel()
				// Final close-frame write is best-effort; connection is going away anyway
				_ = c.conn.Write(ctx, websocket.MessageText, []byte{})
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Write(ctx, websocket.MessageText, message)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		case <-c.hubDone:
			return
		}
	}
}
