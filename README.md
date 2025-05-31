# Go Retry Example

A simple and flexible retry mechanism for Go applications. This package provides a configurable retry mechanism with features like exponential backoff, maximum delay, and custom retry conditions.

## Features

- Configurable number of attempts
- Exponential backoff with configurable factor
- Maximum delay cap
- Optional jitter for delays
- Custom retry conditions
- Context support for cancellation
- Comprehensive test coverage

## Usage

```go
import "retry-example/retry"

// Create a retry configuration
cfg := retry.DefaultConfig()
cfg.Attempts = 3
cfg.Delay = 100 * time.Millisecond
cfg.MaxDelay = 1 * time.Second
cfg.Factor = 2.0

// Define your operation
op := func(ctx context.Context) error {
    // Your operation here
    return nil
}

// Execute with retry
err := retry.Do(context.Background(), cfg, op)
```

## Configuration Options

- `Attempts`: Number of attempts to make (including the first try)
- `Delay`: Initial delay between attempts
- `MaxDelay`: Maximum delay between attempts
- `Factor`: Multiplier for delay after each attempt
- `Jitter`: Whether to add random jitter to delays
- `IsRetryable`: Function to determine if an error is retryable

## License

MIT

## Features Implemented

* Configurable number of retry attempts.
* Configurable initial delay.
* Exponential backoff for subsequent retries.
* Optional jitter to prevent thundering herd scenarios.
* Respects `context.Context` for cancellation and deadlines.
* Ability to define a custom function to determine if an error is retryable.
* Sensible default configuration.

## Structure

* `/retry`: Contains the core retry logic (`retry.go`) and its unit tests (`retry_test.go`).
* `main.go`: Provides example usage scenarios for the retry package.

## How to Run

### Run the Example Application

```bash
go run main.go