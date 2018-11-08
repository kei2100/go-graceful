package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kei2100/go-graceful"
)

func main() {
	srv := http.Server{Handler: mux()}
	go srv.Serve(listener())

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM)
	for range ch {
		srv.Shutdown(context.Background())
		break
	}
}

func mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	mux.Handle("/delay", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(200)
	}))
	return mux
}

func listener() net.Listener {
	lns, err := graceful.InheritedListeners()
	if err != nil {
		panic(err)
	}
	return lns[0]
}
