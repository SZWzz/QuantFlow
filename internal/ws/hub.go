package ws

import (
	"encoding/json"
	"sync"
)

type Message struct {
	Topic string          `json:"topic"`
	Data  json.RawMessage `json:"data"`
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	topics     map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		topics:     make(map[string]map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				for topic := range client.topics {
					if subs, ok := h.topics[topic]; ok {
						delete(subs, client)
					}
				}
				close(client.send)
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Subscribe(client *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*Client]bool)
	}
	h.topics[topic][client] = true
	client.topics[topic] = true
}

func (h *Hub) Unsubscribe(client *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.topics[topic]; ok {
		delete(subs, client)
	}
	delete(client.topics, topic)
}

func (h *Hub) Broadcast(topic string, data any) {
	h.mu.RLock()
	subs := h.topics[topic]
	h.mu.RUnlock()

	raw, err := json.Marshal(data)
	if err != nil {
		return
	}

	msg := &Message{Topic: topic, Data: raw}
	rawMsg, _ := json.Marshal(msg)

	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range subs {
		select {
		case client.send <- rawMsg:
		default:
		}
	}
}
