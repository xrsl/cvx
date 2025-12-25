package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	clog "github.com/xrsl/cvx/pkg/log"
)

// Config holds retry configuration
type Config struct {
	MaxRetries  int           // Maximum number of retry attempts
	BaseDelay   time.Duration // Initial delay between retries
	MaxDelay    time.Duration // Maximum delay between retries
	Multiplier  float64       // Multiplier for exponential backoff
	JitterRatio float64       // Jitter ratio (0-1) to add randomness
}

// DefaultConfig returns sensible defaults for AI API calls
func DefaultConfig() Config {
	return Config{
		MaxRetries:  3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0.1,
	}
}

// RetryableError wraps an error that should trigger a retry
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	var retryable *RetryableError
	return errors.As(err, &retryable)
}

// Retryable wraps an error to indicate it should be retried
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// Do executes the function with retries using exponential backoff
func Do[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) {
			clog.Debug("non-retryable error", "error", err)
			return zero, err
		}

		if attempt == cfg.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := cfg.calculateDelay(attempt)

		clog.Debug("retrying after error",
			"attempt", attempt+1,
			"max_retries", cfg.MaxRetries,
			"delay", delay,
			"error", err,
		)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	// Unwrap the retryable error for final return
	var retryable *RetryableError
	if errors.As(lastErr, &retryable) {
		return zero, retryable.Err
	}
	return zero, lastErr
}

// calculateDelay computes the delay for a given attempt with jitter
func (c Config) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * multiplier^attempt
	delay := float64(c.BaseDelay) * math.Pow(c.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}

	// Add jitter
	if c.JitterRatio > 0 {
		jitter := delay * c.JitterRatio * (rand.Float64()*2 - 1) // -jitter to +jitter
		delay += jitter
	}

	return time.Duration(delay)
}
