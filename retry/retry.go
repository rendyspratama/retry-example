package retry

import (
	"context"
	"fmt"
	"time"
)

// Operation is a function that can be retried.
// It should return nil if successful, or an error if it fails.
type Operation func(ctx context.Context) error

// Config holds the retry configuration
type Config struct {
	// Number of attempts to make (including the first try)
	Attempts int
	// Initial delay between attempts
	Delay time.Duration
	// Maximum delay between attempts
	MaxDelay time.Duration
	// Multiplier for delay after each attempt
	Factor float64
	// Whether to add random jitter to delays
	Jitter bool
	// Function to determine if an error is retryable
	IsRetryable func(error) bool
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Attempts:    3,
		Delay:       100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Factor:      2.0,
		Jitter:      true,
		IsRetryable: func(err error) bool { return true },
	}
}

// Do executes the operation with retries according to the config
func Do(ctx context.Context, cfg Config, op func(context.Context) error) error {
	var lastErr error
	if cfg.IsRetryable == nil {
		cfg.IsRetryable = func(err error) bool { return true }
	}
	delay := cfg.Delay

	for attempt := 1; attempt <= cfg.Attempts; attempt++ {
		// Check context before each attempt
		if err := ctx.Err(); err != nil {
			return err
		}

		// Execute the operation
		err := op(ctx)
		if err == nil {
			return nil // Success!
		}

		lastErr = err

		// Check if error is retryable
		if !cfg.IsRetryable(err) {
			return err
		}

		// If this was the last attempt, don't wait
		if attempt == cfg.Attempts {
			break
		}

		// Calculate next delay
		nextDelay := delay
		if cfg.Factor > 0 {
			nextDelay = time.Duration(float64(delay) * cfg.Factor)
		}
		if cfg.MaxDelay > 0 && nextDelay > cfg.MaxDelay {
			nextDelay = cfg.MaxDelay
		}

		// Wait for next attempt
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}

		delay = nextDelay
	}

	return fmt.Errorf("operation failed after %d attempts: %v", cfg.Attempts, lastErr)
}
