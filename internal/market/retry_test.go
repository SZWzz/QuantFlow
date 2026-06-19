package market

import (
	"errors"
	"testing"
	"time"
)

func TestRetryWithBudget_Success(t *testing.T) {
	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "ok", nil
	}

	config := RetryConfig{
		MaxRetries: 3,
		Backoff:    []time.Duration{1, 1, 1},
		Deadline:   time.Now().Add(5 * time.Second),
		Label:      "test",
	}

	result, err := RetryWithBudget(fn, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestRetryWithBudget_RetryOnTransient(t *testing.T) {
	callCount := 0
	fn := func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", NewTransientError("temporary error")
		}
		return "ok", nil
	}

	config := RetryConfig{
		MaxRetries: 3,
		Backoff:    []time.Duration{1, 1, 1},
		Deadline:   time.Now().Add(5 * time.Second),
		Label:      "test",
	}

	result, err := RetryWithBudget(fn, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetryWithBudget_NonTransientFailsImmediately(t *testing.T) {
	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "", errors.New("permanent error")
	}

	config := RetryConfig{
		MaxRetries: 3,
		Backoff:    []time.Duration{100, 100, 100},
		Deadline:   time.Now().Add(5 * time.Second),
		Label:      "test",
	}

	_, err := RetryWithBudget(fn, config)
	if err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should not retry non-transient)", callCount)
	}
}

func TestRetryWithBudget_ExhaustedRetries(t *testing.T) {
	callCount := 0
	fn := func() (string, error) {
		callCount++
		return "", NewTransientError("always fails")
	}

	config := RetryConfig{
		MaxRetries: 2,
		Backoff:    []time.Duration{1, 1},
		Deadline:   time.Now().Add(5 * time.Second),
		Label:      "test",
	}

	_, err := RetryWithBudget(fn, config)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// Initial call + 2 retries = 3 total calls
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetryWithBudget_DeadlineExceeded(t *testing.T) {
	callCount := 0
	fn := func() (string, error) {
		callCount++
		time.Sleep(10 * time.Millisecond)
		return "", NewTransientError("error")
	}

	config := RetryConfig{
		MaxRetries: 10,
		Backoff:    make([]time.Duration, 10), // all zero
		Deadline:   time.Now().Add(20 * time.Millisecond),
		Label:      "test",
	}

	_, err := RetryWithBudget(fn, config)
	if err == nil {
		t.Fatal("expected deadline error")
	}
}

func TestCheckBudget(t *testing.T) {
	// Past deadline
	err := CheckBudget(time.Now().Add(-1*time.Second), "test")
	if err == nil {
		t.Error("expected error for past deadline")
	}

	// Future deadline
	err = CheckBudget(time.Now().Add(1*time.Hour), "test")
	if err != nil {
		t.Errorf("unexpected error for future deadline: %v", err)
	}
}
