package graceful

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/kei2100/go-graceful/supervisor"
)

// Serve executes given command
// and graceful restarts when the restart signal received.
// default restart signal is HUP.
func Serve(command string, opts ...OptionFunc) error {
	return graceful.Serve(command, opts...)
}

// Restart graceful restarts manually
func Restart() error {
	return graceful.Restart()
}

var graceful = NewGraceful()

// Graceful restart engine
type Graceful struct {
	manualRestartCh   chan struct{}
	manualRestartedCh chan error
}

// NewGraceful creates a new Graceful
func NewGraceful() *Graceful {
	return &Graceful{
		manualRestartCh:   make(chan struct{}),
		manualRestartedCh: make(chan error),
	}
}

// Serve executes given command
// and graceful restarts when the restart signal received.
// default restart signal is HUP.
func (g *Graceful) Serve(command string, opts ...OptionFunc) error {
	o := &option{}
	o.applyOrDefault(opts)

	extraFiles, err := createListenerFiles(o.listeners)
	if err != nil {
		return err
	}
	defer closeListenerFiles(extraFiles)

	sv := &supervisor.Supervisor{
		Command:            command,
		Args:               o.args,
		ExtraFiles:         extraFiles,
		Env:                []string{listenersEnv(o.listeners)},
		WaitReadyFunc:      o.waitReadyFunc,
		AutoRestartEnabled: o.autoRestartEnabled,
		StartTimeout:       o.startTimeout,
		StopOldDelay:       o.stopOldDelay,
	}
	done := make(chan error)
	go func() {
		err := start(sv, o)
		done <- err
		close(done)
	}()

	restartCh := make(chan os.Signal)
	signal.Notify(restartCh, o.restartSignals...)
	shutdownCh := make(chan os.Signal)
	signal.Notify(shutdownCh, o.shutdownSignals...)

	for {
		select {
		case err := <-done:
			return err
		case <-restartCh:
			if err := restart(sv, o); err != nil {
				return err
			}
		case <-g.manualRestartCh:
			err := restart(sv, o)
			g.manualRestartedCh <- err
		case sig := <-shutdownCh:
			if err := shutdown(sv, sig, o); err != nil {
				return err
			}
			return nil
		}
	}
}

// Restart graceful restarts manually
func (g *Graceful) Restart() error {
	g.manualRestartCh <- struct{}{}
	return <-g.manualRestartedCh
}

func start(sv *supervisor.Supervisor, o *option) error {
	ctx, can := o.startContext()
	defer can()
	err := sv.Start(ctx)
	if err != nil {
		return fmt.Errorf("supervisor: failed to start process: %v", err)
	}
	return nil
}

func restart(sv *supervisor.Supervisor, o *option) error {
	ctx, can := o.restartContext()
	defer can()
	err := sv.RestartProcess(ctx, o.gracefulStopSignal)
	if err != nil {
		return fmt.Errorf("graceful: failed to restart process: %v", err)
	}
	return nil
}

func shutdown(sv *supervisor.Supervisor, sig os.Signal, o *option) error {
	ctx, can := o.shutdownContext()
	defer can()
	err := sv.Shutdown(ctx, sig)
	if err != nil {
		return fmt.Errorf("graceful: failed to shutdown process")
	}
	return nil
}
