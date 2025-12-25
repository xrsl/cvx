package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	result, err := Do(ctx, cfg, func() (string, error) {
		calls++
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	result, err := Do(ctx, cfg, func() (string, error) {
		calls++
		if calls < 3 {
			return "", Retryable(errors.New("temporary error"))
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoNonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:  3,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	_, err := Do(ctx, cfg, func() (string, error) {
		calls++
		return "", errors.New("permanent error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for non-retryable), got %d", calls)
	}
}

func TestDoMaxRetries(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:  2,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	_, err := Do(ctx, cfg, func() (string, error) {
		calls++
		return "", Retryable(errors.New("always fails"))
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 3 { // initial + 2 retries
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{
		MaxRetries:  5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	calls := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := Do(ctx, cfg, func() (string, error) {
		calls++
		return "", Retryable(errors.New("keep retrying"))
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestIsRetryable(t *testing.T) {
	if IsRetryable(nil) {
		t.Error("nil should not be retryable")
	}

	if IsRetryable(errors.New("normal error")) {
		t.Error("normal error should not be retryable")
	}

	if !IsRetryable(Retryable(errors.New("retryable"))) {
		t.Error("retryable error should be retryable")
	}
}

func TestRetryableNil(t *testing.T) {
	if Retryable(nil) != nil {
		t.Error("Retryable(nil) should return nil")
	}
}

func TestCalculateDelay(t *testing.T) {
	cfg := Config{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	// Attempt 0: 100ms
	d0 := cfg.calculateDelay(0)
	if d0 != 100*time.Millisecond {
		t.Errorf("attempt 0: expected 100ms, got %v", d0)
	}

	// Attempt 1: 200ms
	d1 := cfg.calculateDelay(1)
	if d1 != 200*time.Millisecond {
		t.Errorf("attempt 1: expected 200ms, got %v", d1)
	}

	// Attempt 2: 400ms
	d2 := cfg.calculateDelay(2)
	if d2 != 400*time.Millisecond {
		t.Errorf("attempt 2: expected 400ms, got %v", d2)
	}
}

func TestCalculateDelayMaxCap(t *testing.T) {
	cfg := Config{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
		Multiplier:  2.0,
		JitterRatio: 0,
	}

	// Attempt 10: would be 100ms * 2^10 = 102.4s, but capped at 500ms
	d := cfg.calculateDelay(10)
	if d != 500*time.Millisecond {
		t.Errorf("expected max delay 500ms, got %v", d)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.BaseDelay != time.Second {
		t.Errorf("expected BaseDelay 1s, got %v", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", cfg.MaxDelay)
	}
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10.0)
	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
	if rl.maxTokens != 10.0 {
		t.Errorf("expected maxTokens 10, got %f", rl.maxTokens)
	}
	if rl.refillRate != 10.0 {
		t.Errorf("expected refillRate 10, got %f", rl.refillRate)
	}
}

func TestRateLimiterWait(t *testing.T) {
	// Create a rate limiter with 10 tokens/second (starts full)
	rl := NewRateLimiter(10.0)
	ctx := context.Background()

	// First 10 waits should be instant (using initial tokens)
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := rl.Wait(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)

	// Should be nearly instant (< 50ms for all 10)
	if elapsed > 50*time.Millisecond {
		t.Errorf("first 10 waits took too long: %v", elapsed)
	}
}

func TestRateLimiterWaitContextCancelled(t *testing.T) {
	// Create a rate limiter with 1 token
	rl := NewRateLimiter(1.0)

	// Drain the initial token
	ctx := context.Background()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("unexpected error draining token: %v", err)
	}

	// Now try with cancelled context (no tokens left)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimiterRefill(t *testing.T) {
	// Create a rate limiter with 10 tokens/second
	rl := NewRateLimiter(10.0)
	ctx := context.Background()

	// Drain all tokens
	for i := 0; i < 10; i++ {
		_ = rl.Wait(ctx)
	}

	// Wait a bit for refill (100ms = 1 token)
	time.Sleep(150 * time.Millisecond)

	// Should be able to get at least 1 token instantly
	start := time.Now()
	if err := rl.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("refill wait took too long: %v", elapsed)
	}
}
