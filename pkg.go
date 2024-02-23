// Package retry provides functionality to retry function calls
package retry

import (
	"context"
	"time"
)

// Config configures the behavior of functions in this package.
type Config struct {
	MaxAttempts int              // if not positive, defaults to single attempt
	RetryOn     func(error) bool // how to check if the error is retryable
	Delay       time.Duration    // optional delay to wait between attempts
}

// Func calls fn at least once and on error retries it according to config values.
// The context is used to stop retry attempts early, the function is called
// at least once even if the context is canceled. Context cancelation is mostly
// useful with a non-zero delay. If the context is canceled, function returns
// an error returned by the Context.Err method.
func Func(ctx context.Context, cfg Config, fn func() error) error {
	if cfg.RetryOn == nil || cfg.MaxAttempts < 1 {
		return fn()
	}
	var err error
retryLoop:
	for i := range cfg.MaxAttempts {
		if i != 0 {
			if cfg.Delay > 0 {
				timer := time.NewTimer(cfg.Delay)
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

// FuncVal calls fn at least once and on error retries it according to config values.
// The context is used to stop retry attempts early, the function is called
// at least once even if the context is canceled. Context cancelation is mostly
// useful with a non-zero delay. If the context is canceled, function returns
// an error returned by the Context.Err method.
func FuncVal[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	if cfg.RetryOn == nil || cfg.MaxAttempts < 1 {
		return fn()
	}
	var err error
	var val T
retryLoop:
	for i := range cfg.MaxAttempts {
		if i != 0 {
			if cfg.Delay > 0 {
				timer := time.NewTimer(cfg.Delay)
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
		val, err = fn()
		if cfg.RetryOn(err) {
			continue
		}
		break
	}
	return val, err
}
