package brokers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIBKRSession_Validate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sso/validate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"token":         "test-session-token",
			"expires":       1800,
		})
	}))
	defer server.Close()

	sess := &ibkrSession{}
	err := sess.validate(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	token := sess.getToken()
	if token != "test-session-token" {
		t.Errorf("getToken() = %q, want %q", token, "test-session-token")
	}
	if !sess.isValid() {
		t.Error("expected session to be valid after validate()")
	}
}

func TestIBKRSession_Validate_NotAuthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
	}))
	defer server.Close()

	sess := &ibkrSession{}
	err := sess.validate(context.Background(), server.Client(), server.URL)
	if err == nil {
		t.Fatal("expected error for not authenticated")
	}
}

func TestIBKRSession_Clear(t *testing.T) {
	sess := &ibkrSession{}
	sess.setToken("tok")
	if !sess.isValid() {
		t.Error("expected valid after set")
	}
	sess.clear()
	if sess.isValid() {
		t.Error("expected invalid after clear")
	}
	if sess.getToken() != "" {
		t.Error("expected empty token after clear")
	}
}

func TestIBKRSession_Invalid_AfterExpiry(t *testing.T) {
	sess := &ibkrSession{}
	sess.mu.Lock()
	sess.token = "tok"
	sess.expiresAt = time.Now().Add(-1 * time.Second) // already expired
	sess.mu.Unlock()
	if sess.isValid() {
		t.Error("expected invalid for expired session")
	}
}
