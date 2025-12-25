// Package signal provides graceful shutdown handling for CLI commands.
package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	clog "github.com/xrsl/cvx/pkg/log"
)

// WithInterrupt returns a context that is cancelled when an interrupt signal
// (SIGINT or SIGTERM) is received. It also returns a cancel function that
// should be called to release resources.
func WithInterrupt(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigCh:
			clog.Debug("received signal", "signal", sig)
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
		close(sigCh)
	}()

	return ctx, cancel
}

// NotifyContext is a convenience wrapper that creates a context cancelled
// on interrupt signals. Unlike WithInterrupt, it doesn't require a parent context.
func NotifyContext() (context.Context, context.CancelFunc) {
	return WithInterrupt(context.Background())
}
