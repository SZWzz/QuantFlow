package ws

import "net/http"

// MarketWSService wraps a ws.Hub as an http.Handler for Wails service Route registration.
// Hub is set during App.ServiceStartup, before the HTTP server starts serving.
type MarketWSService struct {
	Hub *Hub
}

// ServeHTTP implements http.Handler — upgrades HTTP to WebSocket.
func (s *MarketWSService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ServeWS(w, r, s.Hub)
}
