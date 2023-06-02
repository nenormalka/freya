// Package grace implements set of helper functions around syscal.Signals
// for gracefully shutdown and reload the service.
package grace

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ShutdownContext returns child context from passed context which will be canceled
// on incoming signals: SIGINT, SIGTERM, SIGHUP.
// Ends immediately by os.Exit(1) after second signal
func ShutdownContext(c context.Context) context.Context {
	return cancelContextOnSignals(c, true, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
}

// ReloadContext returns child context which will be canceled on syscall.SIGINT or syscall.SIGTERM signal
func StopContext(c context.Context) context.Context {
	return cancelContextOnSignals(c, true, syscall.SIGINT, syscall.SIGTERM)
}

// ReloadContext returns child context which will be canceled on syscall.SIGHUP signal
func ReloadChannel(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	sighups := make(chan struct{})
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGHUP)
		close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				sighups <- struct{}{}
			}
		}
	}()
	<-done
	return sighups
}

func cancelContextOnSignals(c context.Context, osExitOnSecond bool, signals ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(c)
	done := make(chan struct{})
	go listenSignalsFunc(ctx, cancel, done, osExitOnSecond, signals...)()
	<-done
	return ctx
}

func listenSignalsFunc(
	ctx context.Context,
	cancel context.CancelFunc,
	done chan struct{},
	osExitOnSecond bool,
	signals ...os.Signal,
) func() {
	return func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, signals...)
		close(done)

		select {
		case <-ctx.Done():
			return
		case <-ch:
			cancel()
			if osExitOnSecond {
				<-ch
				os.Exit(1)
			}
		}
	}
}
