package graceful

import (
	"context"
	"net"
	"os"
	"syscall"
	"time"
)

// options
type option struct {
	args               []string
	env                []string
	listeners          []net.Listener
	waitReadyFunc      func(ctx context.Context, extraFileConns []net.Conn) error
	autoRestartEnabled bool

	restartSignals     []os.Signal
	shutdownSignals    []os.Signal
	gracefulStopSignal os.Signal

	startTimeout    time.Duration
	shutdownTimeout time.Duration
	restartTimeout  time.Duration
	stopOldDelay    time.Duration
}

func (o *option) applyOrDefault(opts []OptionFunc) {
	o.env = os.Environ()
	o.restartSignals = []os.Signal{syscall.SIGHUP}
	o.shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT}
	o.gracefulStopSignal = syscall.SIGTERM
	o.stopOldDelay = time.Second
	for _, f := range opts {
		f(o)
	}
}

var nopCancelFunc context.CancelFunc = func() {}

func (o *option) startContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	can := nopCancelFunc
	if o.startTimeout > 0 {
		ctx, can = context.WithTimeout(ctx, o.startTimeout)
	}
	return ctx, can
}

func (o *option) shutdownContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	can := nopCancelFunc
	if o.shutdownTimeout > 0 {
		ctx, can = context.WithTimeout(ctx, o.shutdownTimeout)
	}
	return ctx, can
}

func (o *option) restartContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	can := nopCancelFunc
	if o.restartTimeout > 0 {
		ctx, can = context.WithTimeout(ctx, o.restartTimeout)
	}
	return ctx, can
}

// OptionFunc is optional function for graceful
type OptionFunc func(o *option)

// WithArgs set command line arguments
func WithArgs(args ...string) OptionFunc {
	return func(o *option) { o.args = args }
}

// WithEnv set environment variables for worker processes
// Each entry is of the form "key=value".
// If Env is nil, the new process uses the current process's
// environment.
func WithEnv(env ...string) OptionFunc {
	return func(o *option) { o.env = env }
}

// WithListeners set listeners.
// listeners are copied to os.File and set to extra files of worker process.
func WithListeners(listeners ...net.Listener) OptionFunc {
	return func(o *option) { o.listeners = listeners }
}

// WithWaitReadyFunc set WaitReadyFunc
func WithWaitReadyFunc(waitReadyFunc func(context.Context, []net.Conn) error) OptionFunc {
	return func(o *option) { o.waitReadyFunc = waitReadyFunc }
}

// WithAutoRestartEnabled set autoRestartEnabled
func WithAutoRestartEnabled(autoRestartEnabled bool) OptionFunc {
	return func(o *option) {
		o.autoRestartEnabled = autoRestartEnabled
	}
}

// WithTimeout set timeout setting
func WithTimeout(startTimeout, shutdownTimeout, restartTimeout time.Duration) OptionFunc {
	return func(o *option) {
		o.startTimeout = startTimeout
		o.shutdownTimeout = shutdownTimeout
		o.restartTimeout = restartTimeout
	}
}

// WithStopOldDelay set stopOldDelay
func WithStopOldDelay(stopOldDelay time.Duration) OptionFunc {
	return func(o *option) { o.stopOldDelay = stopOldDelay }
}
