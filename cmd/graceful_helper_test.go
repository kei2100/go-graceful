package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// gp represents the "graceful" process
type gp struct {
	cmd        *exec.Cmd
	listenAddr string
}

func startGraceful() (*gp, error) {
	addr, err := freeTCPAddr()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("./graceful", "-l", addr, "--", "./stub_http")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &gp{cmd: cmd, listenAddr: addr}, nil
}

func (g *gp) stopGraceful(timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		_, err := g.cmd.Process.Wait()
		done <- err
	}()
	g.cmd.Process.Signal(syscall.SIGTERM)
	ctx, can := context.WithTimeout(context.Background(), timeout)
	defer can()
	select {
	case <-ctx.Done():
		return fmt.Errorf("an error occurred while waiting for stop graceful: %v", ctx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("an error occurred while waiting for stop graceful: %v", err)
		}
	}
	return nil
}

func (g *gp) restartGraceful() error {
	if err := g.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send sighup: %v", err)
	}
	return nil
}

func freeTCPAddr() (string, error) {
	ln, err := net.Listen("tcp", "localhost:")
	if err != nil {
		return "", fmt.Errorf("failed to get free port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().String(), nil
}
