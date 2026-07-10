package market

import (
	"context"
	"fmt"
	"time"
)

// DefaultBackoff is the default sleep schedule for retry attempts.
// Index 0 = sleep before retry 1, index 1 = before retry 2, etc.
var DefaultBackoff = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond, 4 * time.Second}

// RetryConfig configures the retry behavior.
type RetryConfig struct {
	MaxRetries int
	Backoff    []time.Duration
	Deadline   time.Time
	Label      string // free-form label for error messages
}

// DefaultRetryConfig returns a sensible retry configuration.
func DefaultRetryConfig(label string) RetryConfig {
	d := time.Now().Add(30 * time.Second)
	return RetryConfig{
		MaxRetries: 3,
		Backoff:    DefaultBackoff,
		Deadline:   d,
		Label:      label,
	}
}

// CheckBudget returns an error if the deadline has been exceeded.
// Use this between pages of a paginated fetch to fail fast.
func CheckBudget(deadline time.Time, label string) error {
	if time.Now().After(deadline) {
		return fmt.Errorf("%s: deadline exceeded", label)
	}
	return nil
}

// RetryWithBudget executes fn with bounded retries on transient errors.
// Between attempts it sleeps min(backoff[i], remaining time before deadline).
// Any error from fn that implements the TransientError interface is retried.
// Non-transient errors propagate immediately.
func RetryWithBudget[T any](fn func() (T, error), config RetryConfig) (T, error) {
	var zero T

	if len(config.Backoff) < config.MaxRetries {
		return zero, fmt.Errorf("backoff has %d entries; need >= maxRetries (%d)", len(config.Backoff), config.MaxRetries)
	}

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check deadline before each attempt
		if err := CheckBudget(config.Deadline, config.Label); err != nil {
			return zero, err
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		// Check if error is transient
		if te, ok := err.(TransientError); ok && te.IsTransient() {
			if attempt == config.MaxRetries {
				return zero, fmt.Errorf("%s: all %d retries exhausted: %w", config.Label, config.MaxRetries+1, err)
			}

			remaining := time.Until(config.Deadline)
			if remaining <= 0 {
				return zero, fmt.Errorf("%s: deadline exceeded after %d attempts: %w", config.Label, attempt+1, err)
			}

			sleepDuration := config.Backoff[attempt]
			if sleepDuration > remaining {
				sleepDuration = remaining
			}
			time.Sleep(sleepDuration)
			continue
		}

		// Non-transient error: return immediately
		return zero, err
	}

	return zero, fmt.Errorf("%s: unreachable", config.Label)
}

// TransientError is implemented by errors that should trigger a retry.
// Network timeouts, rate limits, and temporary API failures are transient.
// Authentication failures, invalid symbols, and permanent errors are not.
type TransientError interface {
	error
	IsTransient() bool
}

// transientError is a simple implementation of TransientError.
type transientError struct {
	msg string
}

func (e *transientError) Error() string    { return e.msg }
func (e *transientError) IsTransient() bool { return true }

// NewTransientError creates a new transient error (will be retried).
func NewTransientError(msg string) error {
	return &transientError{msg: msg}
}

// NewTransientErrorf creates a formatted transient error.
func NewTransientErrorf(format string, args ...any) error {
	return &transientError{msg: fmt.Sprintf(format, args...)}
}

// RequestCtx returns a context with a 30-second timeout for market data
// requests. Callers must call the returned cancel function to release resources.
func RequestCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
