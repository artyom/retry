// Package retry provides functionality to retry function calls
package retry

import (
	"context"
	"time"
)

// Config configures the behavior of functions in this package.
type Config struct {
	// MaxAttempts specifies the maximum number of retry attempts.
	// If not positive, it defaults to a single attempt (no retries).
	MaxAttempts int
	// RetryOn is a function that determines whether an error is retryable.
	// It should return true if the error is retryable, false otherwise.
	RetryOn func(error) bool
	// Delay specifies a fixed delay between retry attempts.
	// Use WithDelayFunc to implement more complex retry strategies.
	Delay time.Duration

	delayFn func(int) time.Duration
}

// WithDelayFunc returns a copy of the [Config] with a custom delay function.
//
// The provided function is called with the retry attempt number (starting at 1)
// and should return the delay duration for that attempt.
//
// This allows implementing custom backoff strategies.
func (c *Config) WithDelayFunc(fn func(int) time.Duration) Config {
	cfg := *c
	cfg.delayFn = fn
	return cfg
}

// Func retries the provided function according to the [Config].
// It returns the error from the last attempt, or nil on success.
// The provided context can be used to cancel retries early.
//
// If the context is canceled, function returns an error returned
// by the Context.Err method.
func Func(ctx context.Context, cfg Config, fn func() error) error {
	if cfg.RetryOn == nil || cfg.MaxAttempts < 1 {
		return fn()
	}
	var err error
retryLoop:
	for i := range cfg.MaxAttempts {
		if i != 0 {
			if cfg.Delay > 0 || cfg.delayFn != nil {
				delay := cfg.Delay
				if cfg.delayFn != nil {
					delay = max(0, cfg.delayFn(i))
				}
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					err = ctx.Err()
					break retryLoop
				case <-timer.C:
				}
			} else {
				select {
				case <-ctx.Done():
					err = ctx.Err()
					break retryLoop
				default:
				}
			}
		}
		err = fn()
		if cfg.RetryOn(err) {
			continue
		}
		break
	}
	return err
}

// FuncVal retries the provided function according to the [Config].
// It returns the function result and error from the last attempt.
// The provided context can be used to cancel retries early.
//
// If the context is canceled, function returns an error returned
// by the Context.Err method.
func FuncVal[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var val T
	wrap := func() error {
		var err error
		val, err = fn()
		return err
	}
	err := Func(ctx, cfg, wrap)
	return val, err
}
