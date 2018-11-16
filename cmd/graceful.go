package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/kei2100/go-graceful"
	"github.com/spf13/pflag"
)

var (
	listens            []string
	env                []string
	autoRestartEnabled bool
	startTimeout       time.Duration
	shutdownTimeout    time.Duration
	restartTimeout     time.Duration
	stopOldDelay       time.Duration
	help               bool

	// TODO
	//restartSignals     []os.Signal
	//shutdownSignals    []os.Signal
	//gracefulStopSignal os.Signal
)

func init() {
	pflag.Usage = func() {
		name := "graceful"
		fmt.Fprintf(os.Stderr, "The %s provides graceful terminate and restart for socket-based servers\n\n", name)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [flags] -- <command> [args...]\n\n", name)
		fmt.Fprintf(os.Stderr, "Flags:\n")
		pflag.PrintDefaults()
	}
	pflag.StringSliceVarP(&listens, "listen", "l", []string{}, "listen tcp address(es). e.g. -l 127.0.0.1:8000 -l 127.0.0.1:8001")
	pflag.StringSliceVarP(&env, "env", "e", []string{}, "additional environment variables. e.g. -e AAA=BBB -e CCC=DDD")
	pflag.BoolVar(&autoRestartEnabled, "auto-restart-enabled", false, "specifies if the graceful should automatically restart a worker if the worker process exits")
	pflag.DurationVar(&startTimeout, "start-timeout", 10*time.Second, "amount of time the graceful will wait for the worker started")
	pflag.DurationVar(&shutdownTimeout, "shutdown-timeout", 10*time.Second, "amount of time the graceful will wait for the worker shutdown")
	pflag.DurationVar(&restartTimeout, "restart-timeout", 20*time.Second, "amount of time the graceful will wait for the worker restarted")
	pflag.DurationVar(&stopOldDelay, "stop-old-delay", time.Second, "amount of time to suspend to the old worker shutdown")
	pflag.BoolVarP(&help, "help", "h", false, "show this help")
}

func main() {
	pflag.Parse()
	args := pflag.Args()
	if help || len(args) < 1 {
		pflag.Usage()
		os.Exit(2)
	}
	env = append(os.Environ(), env...)
	lns, err := createListeners()
	if err != nil {
		log.Fatalln(err)
	}
	defer closeListeners(lns)

	err = graceful.Serve(
		args[0],
		graceful.WithArgs(args[1:]...),
		graceful.WithEnv(env...),
		graceful.WithListeners(lns...),
		graceful.WithAutoRestartEnabled(autoRestartEnabled),
		graceful.WithTimeout(startTimeout, restartTimeout, shutdownTimeout),
		graceful.WithStopOldDelay(stopOldDelay),
	)
	if err != nil {
		log.Fatalln(err)
	}
}

func createListeners() ([]net.Listener, error) {
	lns := make([]net.Listener, 0)
	for _, addr := range listens {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("main: failed to create a lister %s: %v", addr, err)
		}
		lns = append(lns, ln)
	}
	return lns, nil
}

func closeListeners(lns []net.Listener) {
	for _, ln := range lns {
		if err := ln.Close(); err != nil {
			log.Printf("main: an error occurred when close the listener :%v", err)
		}
	}
}
