package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"retry-example/retry"
)

// A sample operation that might fail a few times before succeeding.
func simulatedNetworkCall(ctx context.Context, callID int) error {
	log.Printf("[Call %d] Attempting simulated network call...", callID)

	// Simulate some work/latency
	select {
	case <-time.After(50 * time.Millisecond): // Simulate some processing time
	case <-ctx.Done():
		log.Printf("[Call %d] Context cancelled during simulated work.", callID)
		return ctx.Err()
	}

	// Simulate a chance of failure
	// #nosec G404
	if rand.Intn(10) < 7 { // ~70% chance of failure for demonstration
		err := errors.New("simulated network error: connection timeout")
		log.Printf("[Call %d] Operation failed: %v", callID, err)
		return err
	}

	log.Printf("[Call %d] Operation succeeded!", callID)
	return nil
}

// A specific error type to demonstrate non-retryable errors
var ErrBadRequest = errors.New("bad request, do not retry")

func main() {
	// Seed random number generator (once at startup)
	// #nosec G404
	rand.New(rand.NewSource(time.Now().UnixNano()))
	log.Println("--- Example 1: Default Retry Config ---")
	call1Attempt := 0
	op1 := func(ctx context.Context) error {
		call1Attempt++
		return simulatedNetworkCall(ctx, call1Attempt)
	}

	cfg1 := retry.DefaultConfig()
	cfg1.Attempts = 5 // Try a bit more for demo
	err1 := retry.Do(context.Background(), cfg1, op1)

	if err1 != nil {
		log.Printf("Main: Example 1 ultimately failed: %v\n", err1)
	} else {
		log.Println("Main: Example 1 eventually succeeded.")
	}

	log.Println("\n--- Example 2: Retry with Context Timeout ---")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond) // Short overall timeout
	defer cancel()

	call2Attempt := 0
	op2 := func(c context.Context) error {
		call2Attempt++
		// This operation itself could be slow, leading to context timeout during retry waits or execution
		return simulatedNetworkCall(c, call2Attempt)
	}

	cfg2 := retry.DefaultConfig()
	cfg2.Attempts = 10                 // Allow many attempts
	cfg2.Delay = 70 * time.Millisecond // Longer individual delay to hit timeout faster
	err2 := retry.Do(ctx, cfg2, op2)

	if err2 != nil {
		log.Printf("Main: Example 2 ultimately failed: %v\n", err2)
		if errors.Is(err2, context.DeadlineExceeded) {
			log.Println("Main: Example 2 failed due to context deadline exceeded, as expected.")
		}
	} else {
		log.Println("Main: Example 2 eventually succeeded (unexpected with short timeout).")
	}

	log.Println("\n--- Example 3: Non-Retryable Error ---")
	call3Attempt := 0
	op3 := func(ctx context.Context) error {
		call3Attempt++
		if call3Attempt == 2 { // Fail with a non-retryable error on the 2nd attempt
			log.Printf("[Call %d] Operation returning a non-retryable error.", call3Attempt)
			return ErrBadRequest
		}
		log.Printf("[Call %d] Operation returning a generic retryable error.", call3Attempt)
		return fmt.Errorf("generic retryable error attempt %d", call3Attempt)
	}

	cfg3 := retry.DefaultConfig()
	cfg3.Attempts = 5
	cfg3.IsRetryable = func(err error) bool {
		return !errors.Is(err, ErrBadRequest) // Don't retry on ErrBadRequest
	}
	err3 := retry.Do(context.Background(), cfg3, op3)
	if err3 != nil {
		log.Printf("Main: Example 3 ultimately failed: %v\n", err3)
		if strings.Contains(err3.Error(), ErrBadRequest.Error()) {
			log.Println("Main: Example 3 failed due to non-retryable error, as expected.")
		}
	} else {
		log.Println("Main: Example 3 eventually succeeded (should not happen with non-retryable error).")
	}
}
