package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ibkrSession manages the IBKR Client Portal session token and expiry.
type ibkrSession struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	stopCh    chan struct{}
}

// isValid checks if the current session is still valid locally.
func (s *ibkrSession) isValid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token != "" && time.Now().Before(s.expiresAt)
}

// setToken stores the session token with a 30-minute expiry.
func (s *ibkrSession) setToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
	s.expiresAt = time.Now().Add(30 * time.Minute)
}

// getToken returns the current session token.
func (s *ibkrSession) getToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// clear resets the session.
func (s *ibkrSession) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
	s.expiresAt = time.Time{}
}

// startRefresh launches a background goroutine that validates the session every 4 minutes.
func (s *ibkrSession) startRefresh(ctx context.Context, client *http.Client, baseURL string) {
	s.stopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(4 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.validate(ctx, client, baseURL); err != nil {
					slog.Warn("ibkr session refresh failed", "error", err)
				}
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// stopRefresh stops the background session refresh goroutine.
func (s *ibkrSession) stopRefresh() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
}

// validate performs GET /sso/validate to check if the session is still valid.
func (s *ibkrSession) validate(ctx context.Context, client *http.Client, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/sso/validate", nil)
	if err != nil {
		return fmt.Errorf("ibkr session validate: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ibkr session validate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ibkr session validate: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Authenticated bool   `json:"authenticated"`
		Token         string `json:"token"`
		Expires       int    `json:"expires"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("ibkr session validate parse: %w", err)
	}
	if !result.Authenticated {
		return fmt.Errorf("ibkr session not authenticated — user must log into IB Gateway")
	}

	s.setToken(result.Token)
	slog.Debug("ibkr session refreshed", "expires_in_s", result.Expires)
	return nil
}
