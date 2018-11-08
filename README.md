go-graceful
===

The graceful provides graceful terminate and restart for socket-based servers written in Golang

### Example

```go
// http_server.go
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
		// serve http
		lns, _ := graceful.InheritedListeners()
		srv.Serve(lns[0])
		close(done)
	}()

	// shutdown http server when receive the SIGTERM
	<-ch
	srv.Shutdown(context.Background())
	<-done
}
```

```bash
$ go build http_server.go && ./http_server
```

```bash
$ ps aux | grep http_serve[r]
kei2100        53253   0.0  0.0  4393128   7804 s006  S+   11:15AM   0:00.02 ./http_server
kei2100        53254   0.0  0.0  4389140   7692 s006  S+   11:15AM   0:00.01 /Users/kei2100/go/src/github.com/kei2100/go-graceful/example/http_server

# 53253 is supervisor process
# 53254is  worker process

$ lsof -i:8080
COMMAND     PID      USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
http_serv 53253 kei2100    5u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)
http_serv 53253 kei2100    7u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)
http_serv 53254 kei2100    3u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)
http_serv 53254 kei2100    6u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)

$ curl localhost:8080
OK
```

Graceful restart worker process

```bash
$ kill -s HUP 53253

$ ps aux | grep http_serve[r]
kei2100        53253   0.0  0.0  4393128   7804 s006  S+   11:15AM   0:00.02 ./http_server
kei2100        54001   0.0  0.0  4389140   7692 s006  S+   11:17AM   0:00.01 /Users/kei2100/go/src/github.com/kei2100/go-graceful/example/http_server

# 54001 is restarted worker processc

$ lsof -i:8080
COMMAND     PID      USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
http_serv 53253 kei.arima    5u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)
http_serv 53253 kei.arima    7u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)
http_serv 54001 kei.arima    3u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)
http_serv 54001 kei.arima    6u  IPv4 0x72f2723fa756899b      0t0  TCP localhost:http-alt (LISTEN)

$ curl localhost:8080
OK
```
