package retry_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/artyom/retry"
)

func ExampleFuncVal() {
	var n int
	fn := func() (int, error) {
		n++
		if n < 3 {
			fmt.Printf("attempt %d, failing\n", n)
			return 0, errors.New("boom")
		}
		return n, nil
	}
	cfg := retry.Config{
		MaxAttempts: 10,
		RetryOn:     func(err error) bool { return err != nil },
	}
	val, err := retry.FuncVal(context.Background(), cfg, fn)
	fmt.Printf("val: %d, error: %v\n", val, err)
	// Output:
	// attempt 1, failing
	// attempt 2, failing
	// val: 3, error: <nil>
}

func ExampleFunc() {
	var n int
	fn := func() error {
		n++
		if n < 3 {
			fmt.Printf("attempt %d, failing\n", n)
			return errors.New("boom")
		}
		fmt.Printf("attempt %d, succeeding\n", n)
		return nil
	}
	cfg := retry.Config{
		MaxAttempts: 2,
		RetryOn:     func(err error) bool { return err != nil },
	}
	err := retry.Func(context.Background(), cfg, fn)
	fmt.Println("error:", err)

	// reset fn state
	n = 0
	// adjust config to do more attempts
	cfg.MaxAttempts = 10

	fmt.Println()
	fmt.Println("after adjustments:")
	err = retry.Func(context.Background(), cfg, fn)
	fmt.Println("error:", err)
	// Output:
	// attempt 1, failing
	// attempt 2, failing
	// error: boom
	//
	// after adjustments:
	// attempt 1, failing
	// attempt 2, failing
	// attempt 3, succeeding
	// error: <nil>
}

func TestFunc(t *testing.T) {
	t.Run("delay", func(t *testing.T) {
		var runDelays [3]time.Duration
		cfg := retry.Config{
			MaxAttempts: len(runDelays),
			Delay:       5 * time.Millisecond,
			RetryOn:     func(err error) bool { return err != nil },
		}
		begin := time.Now()
		var i int
		fn := func() error {
			runDelays[i] = time.Since(begin).Round(cfg.Delay)
			i++
			return errors.New("boom")
		}
		err := retry.Func(context.Background(), cfg, fn)
		if err == nil {
			t.Fatal("expected to get error from retry.Func, but got nil")
		}
		if d := time.Since(begin).Round(cfg.Delay); d != 10*time.Millisecond {
			t.Fatalf("total time took %v, instead of 10ms", d)
		}
		want := [3]time.Duration{0, 5 * time.Millisecond, 10 * time.Millisecond}
		if want != runDelays {
			t.Fatalf("got wrong delays: %v, want %v", runDelays, want)
		}
	})
	t.Run("cancel", func(t *testing.T) {
		fn := func() error { return errors.New("boom") }
		cfg := retry.Config{
			MaxAttempts: 10,
			RetryOn:     func(err error) bool { return err != nil },
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := retry.Func(ctx, cfg, fn)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got unexpected error %v, want %v", err, context.Canceled)
		}
	})
}

func ExampleConfig_WithDelayFunc() {
	cfg := retry.Config{
		MaxAttempts: 3,
		RetryOn:     func(err error) bool { return err != nil },
	}
	fn := func() error { return errors.New("always failing") }

	delayFunc := func(i int) time.Duration {
		fmt.Println("delayFunc called with argument", i)
		return time.Millisecond * time.Duration(i) // increase delay with each attempt made
	}
	// retry.Func call below calls fn() over 3 attempts, calling delayFunc twice:
	//	- fn() call
	//	- delayFunc(1) call
	//	- fn() call
	//	- delayFunc(2) call
	//	- fn() call
	err := retry.Func(context.Background(), cfg.WithDelayFunc(delayFunc), fn)
	fmt.Println("error:", err)
	// Output:
	// delayFunc called with argument 1
	// delayFunc called with argument 2
	// error: always failing
}
