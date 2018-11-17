package worker

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Worker represents a worker process
type Worker struct {
	Command       string
	Args          []string
	ExtraFiles    []*os.File
	Env           []string
	WaitReadyFunc func(ctx context.Context, extraFileConns []net.Conn) error
	StartTimeout  time.Duration

	autoRestart   bool
	autoRestartMu sync.RWMutex

	cmd   *exec.Cmd
	cmdMu sync.RWMutex
	stop  chan struct{}
}

// Start this Worker
func (w *Worker) Start(ctx context.Context) error {
	if w.WaitReadyFunc == nil {
		w.WaitReadyFunc = func(_ context.Context, _ []net.Conn) error { return nil }
	}
	if err := w.startProcess(ctx); err != nil {
		return err
	}
	w.stop = make(chan struct{})
	go func() {
		defer close(w.stop)
		for {
			if err := w.waitProcess(); err != nil {
				log.Println(err)
			}
			if !w.isAutoRestart() {
				return
			}
			log.Println("worker: auto restarting")
			if err := w.startProcess(context.Background()); err != nil {
				log.Println(err)
			}
		}
	}()
	return nil
}

// Stop this Worker
func (w *Worker) Stop(ctx context.Context, sig os.Signal) error {
	w.SetAutoRestart(false)
	if err := w.signalProcess(sig); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("worker: an error occurred while waiting for stop: %v", ctx.Err())
	case <-w.stop:
		return nil
	}
}

// Kill causes the Worker process to exit immediately.
// Kill does not wait until the Process has actually exited.
func (w *Worker) Kill() error {
	w.SetAutoRestart(false)
	return w.signalProcess(os.Kill)
}

// Done returns a channel that's closed when this worker is done
func (w *Worker) Done() <-chan struct{} {
	return w.stop
}

// SetAutoRestart set autoRestartEnabled
func (w *Worker) SetAutoRestart(enabled bool) {
	w.autoRestartMu.Lock()
	defer w.autoRestartMu.Unlock()
	w.autoRestart = enabled
}

func (w *Worker) isAutoRestart() bool {
	w.autoRestartMu.RLock()
	defer w.autoRestartMu.RUnlock()
	return w.autoRestart
}

func (w *Worker) startProcess(ctx context.Context) error {
	var can context.CancelFunc = func() {}
	if w.StartTimeout > 0 {
		ctx, can = context.WithTimeout(ctx, w.StartTimeout)
	}
	defer can()

	w.cmdMu.Lock() // cmd LOCK
	cmd := exec.Command(w.Command, w.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = w.ExtraFiles
	cmd.Env = w.Env
	if err := cmd.Start(); err != nil {
		w.cmdMu.Unlock() // cmd UNLOCK
		return fmt.Errorf("worker: failed to restart command: %v", err)
	}
	w.cmd = cmd
	w.cmdMu.Unlock() // cmd UNLOCK

	conns, err := createFileConns(w.cmd.ExtraFiles)
	if err != nil {
		return err
	}
	defer closeFileConns(conns)
	if err := w.WaitReadyFunc(ctx, conns); err != nil {
		return fmt.Errorf("worker: WaitReadyFunc returns %v", err)
	}
	return nil
}

func (w *Worker) waitProcess() error {
	w.cmdMu.RLock() // cmd LOCK
	cmd := w.cmd
	w.cmdMu.RUnlock() // cmd UNLOCK

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("worker: command abnormally finished: %v", err)
	}
	return nil
}

func (w *Worker) signalProcess(sig os.Signal) error {
	w.cmdMu.RLock()
	defer w.cmdMu.RUnlock()

	if err := w.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("worker: failed to send %s: %v", sig, err)
	}
	return nil
}

func createFileConns(files []*os.File) ([]net.Conn, error) {
	conns := make([]net.Conn, 0)
	for _, f := range files {
		c, err := net.FileConn(f)
		if err != nil {
			closeFileConns(conns)
			return nil, fmt.Errorf("worker: failed to create file connection: %v", err)
		}
		conns = append(conns, &onceCloseConn{Conn: c})
	}
	return conns, nil
}

func closeFileConns(conns []net.Conn) {
	for _, c := range conns {
		if err := c.Close(); err != nil {
			log.Printf("worker: close file connection error: %v", err)
		}
	}
}

type onceCloseConn struct {
	once sync.Once
	net.Conn
}

func (c *onceCloseConn) Close() error {
	var err error
	c.once.Do(func() {
		err = c.Conn.Close()
	})
	return err
}
