go-graceful
===

The graceful provides graceful terminate and restart for socket-based servers written in Golang

```
Usage:
  graceful [flags] -- <command> [args...]

Flags:
      --auto-restart-enabled        specifies if the graceful should automatically restart a worker if the worker process exits
  -h, --help                        show this help
  -l, --listen strings              listen tcp address(es). e.g. -l 127.0.0.1:8000 -l 127.0.0.1:8001
      --restart-timeout duration    amount of time the graceful will wait for the worker restarted (default 20s)
      --shutdown-timeout duration   amount of time the graceful will wait for the worker shutdown (default 10s)
      --start-timeout duration      amount of time the graceful will wait for the worker started (default 10s)
      --stop-old-delay duration     amount of time to suspend to the old worker shutdown (default 1s)
```
