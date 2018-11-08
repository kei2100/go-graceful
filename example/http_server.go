package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kei2100/go-graceful"
)

func main() {
	// supervisor process
	if _, ok := os.LookupEnv("WORKER"); !ok {
		command, _ := os.Executable()
		ln, _ := net.Listen("tcp", "localhost:8080")
		// launch worker
		err := graceful.Serve(
			command,
			graceful.WithListeners(ln),
			graceful.WithEnv("WORKER=true"),
		)
		if err != nil {
			log.Println(err)
		}
		return
	}

	// worker process
	srv := http.Server{}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK\n"))
	})
	done := make(chan struct{})
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM)

	go func() {
		lns, _ := graceful.InheritedListeners()
		srv.Serve(lns[0])
		close(done)
	}()

	<-ch
	srv.Shutdown(context.Background())
	<-done
}
