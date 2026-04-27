package ffuf

import (
	"context"
	"testing"
)

// Regression: interruptMonitor's signal-handler goroutine called
// pauseWg.Done() based on j.Paused but never cleared j.Paused. A second
// SIGINT arriving while the job was still paused (a common pattern when the
// user mashes Ctrl-C) re-entered the same branch and called Done() on a
// counter that was already 0, panicking inside the goroutine. Because the
// goroutine had no recover, the panic killed the whole ffuf process.
func TestJobHandleInterruptIdempotentWhilePaused(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conf := NewConfig(ctx, cancel)
	conf.Threads = 1
	j := NewJob(&conf)
	j.Output = NewNullOutput()

	j.Pause() // pauseWg.Add(1)

	// First signal: matches the Pause; pauseWg counter goes 1 -> 0.
	j.handleInterrupt()

	// Second signal arriving before any Resume must not blow up. With the
	// pre-fix code the WaitGroup counter would underflow and panic from
	// inside the (unrecovered) signal-handler goroutine.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handleInterrupt panicked on second invocation while still paused: %v", r)
		}
	}()
	j.handleInterrupt()
}
