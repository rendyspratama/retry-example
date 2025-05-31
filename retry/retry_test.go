package retry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestSuccessOnFirstAttempt checks if the operation succeeds immediately.
func TestSuccessOnFirstAttempt(t *testing.T) {
	op := func(ctx context.Context) error {
		return nil // Success!
	}
	cfg := DefaultConfig()
	cfg.Attempts = 3

	err := Do(context.Background(), cfg, op)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

// TestSuccessAfterRetries checks if the operation succeeds after a few failures.
func TestSuccessAfterRetries(t *testing.T) {
	attemptsMade := 0
	maxFailures := 2 // Fail twice, succeed on the 3rd attempt

	op := func(ctx context.Context) error {
		attemptsMade++
		if attemptsMade <= maxFailures {
			return fmt.Errorf("simulated failure %d", attemptsMade)
		}
		return nil // Success!
	}

	cfg := DefaultConfig()
	cfg.Attempts = 3
	cfg.Delay = 1 * time.Millisecond // Keep test fast

	err := Do(context.Background(), cfg, op)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
	if attemptsMade != maxFailures+1 {
		t.Errorf("Expected %d attempts, but made %d", maxFailures+1, attemptsMade)
	}
}

// TestFailureAfterAllAttempts checks if an error is returned after all attempts fail.
func TestFailureAfterAllAttempts(t *testing.T) {
	attemptsMade := 0
	op := func(ctx context.Context) error {
		attemptsMade++
		return fmt.Errorf("persistent simulated failure %d", attemptsMade)
	}

	cfg := DefaultConfig()
	cfg.Attempts = 3
	cfg.Delay = 1 * time.Millisecond

	err := Do(context.Background(), cfg, op)
	if err == nil {
		t.Errorf("Expected an error, but got nil")
	} else {
		if !strings.Contains(err.Error(), "persistent simulated failure 3") { // Check underlying error
			t.Errorf("Error message mismatch. Expected to contain '%s', got '%s'", "persistent simulated failure 3", err.Error())
		}
		if !strings.HasPrefix(err.Error(), "operation failed after 3 attempts") {
			t.Errorf("Error prefix mismatch. Expected prefix for ultimate failure, got '%s'", err.Error())
		}
	}

	if attemptsMade != cfg.Attempts {
		t.Errorf("Expected %d attempts, but made %d", cfg.Attempts, attemptsMade)
	}
}

// TestContextCancellation checks if retries stop when context is cancelled.
func TestContextCancellation(t *testing.T) {
	attemptsMade := 0
	op := func(ctx context.Context) error {
		attemptsMade++
		// Simulate an operation that takes some time, allowing context to be cancelled
		select {
		case <-time.After(50 * time.Millisecond):
			return errors.New("simulated failure, should have been cancelled")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	cfg := DefaultConfig()
	cfg.Attempts = 5
	cfg.Delay = 10 * time.Millisecond // Short delay for testing

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond) // Cancel context quickly
	defer cancel()

	err := Do(ctx, cfg, op)

	if err == nil {
		t.Fatal("Expected an error due to context cancellation, but got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.DeadlineExceeded or context.Canceled, got %v", err)
	}

	// We expect fewer than max attempts because cancellation should stop it early.
	// It's hard to be exact due to timing, but it should not reach cfg.Attempts.
	if attemptsMade >= cfg.Attempts {
		t.Errorf("Expected fewer than %d attempts due to cancellation, got %d", cfg.Attempts, attemptsMade)
	}
	if attemptsMade == 0 {
		t.Error("Expected at least one attempt before cancellation, got zero")
	}
	t.Logf("Attempts made before cancellation: %d", attemptsMade)
}

// TestNonRetryableError checks if retries stop for a non-retryable error.
func TestNonRetryableError(t *testing.T) {
	var ErrNonRetryable = errors.New("this error is not retryable")
	attemptsMade := 0

	op := func(ctx context.Context) error {
		attemptsMade++
		if attemptsMade == 1 {
			return ErrNonRetryable
		}
		return errors.New("this should not be reached")
	}

	cfg := DefaultConfig()
	cfg.Attempts = 3
	cfg.IsRetryable = func(err error) bool {
		return !errors.Is(err, ErrNonRetryable)
	}

	err := Do(context.Background(), cfg, op)
	if err == nil {
		t.Fatal("Expected a non-retryable error, but got nil")
	}
	if !strings.Contains(err.Error(), ErrNonRetryable.Error()) {
		t.Errorf("Expected error to be '%v', got '%v'", ErrNonRetryable, err)
	}
	if attemptsMade != 1 {
		t.Errorf("Expected only 1 attempt for a non-retryable error, got %d", attemptsMade)
	}
}

// TestExponentialBackoffAndMaxDelay checks if delays increase and respect MaxDelay
func TestExponentialBackoffAndMaxDelay(t *testing.T) {
	// This test is more about observing logs than strict pass/fail on timing in a unit test.
	// Precise timing tests are tricky in Go's default test runner.
	// We'll check if the number of attempts is as expected.
	t.Log("Testing exponential backoff (inspect logs for delay increases)")
	attemptsMade := 0
	op := func(ctx context.Context) error {
		attemptsMade++
		return fmt.Errorf("failure %d", attemptsMade)
	}

	cfg := Config{
		Attempts: 4,
		Delay:    5 * time.Millisecond,
		MaxDelay: 20 * time.Millisecond,
		Factor:   2.0,
		Jitter:   false, // Disable jitter for predictable delay calculation in logs
	}

	err := Do(context.Background(), cfg, op)
	if err == nil {
		t.Error("Expected error after all attempts, got nil")
	}
	if attemptsMade != cfg.Attempts {
		t.Errorf("Expected %d attempts, got %d", cfg.Attempts, attemptsMade)
	}
	// To verify delays, you'd typically inspect the logs printed by retry.Do
	// Expected delays (approx):
	// Attempt 1: no delay
	// Attempt 2: wait ~5ms
	// Attempt 3: wait ~10ms (5 * 2)
	// Attempt 4: wait ~20ms (10 * 2, capped by MaxDelay)
}
