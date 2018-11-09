package graceful

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const envKey = "GRACEFUL_LISTENERS"
const envSep = ";"

// InheritedListeners creates listeners from fd.
// this func only for worker process
func InheritedListeners() ([]net.Listener, error) {
	lns := make([]net.Listener, 0)
	envVal := os.Getenv(envKey)
	for i, addr := range strings.Split(envVal, envSep) {
		ln, err := func() (net.Listener, error) {
			fd := uintptr(3 + i) // 0:stdin, 1:stdout, 2:stderr
			f := os.NewFile(fd, addr)
			if f == nil {
				return nil, fmt.Errorf("graceful: failed to NewFile. fd %v, addr %v", fd, addr)
			}
			defer f.Close()
			ln, err := net.FileListener(f)
			if err != nil {
				return nil, fmt.Errorf("graceful: failed to create file listener: %v", err)
			}
			return ln, nil
		}()
		if err != nil {
			return nil, err
		}
		lns = append(lns, ln)
	}
	return lns, nil
}

// listenersEnv returns env var from listener addrs.
// e.g. GRACEFUL_LISTENERS=127.0.0.1:8080;127.0.0.1:8081
// this func only for supervisor process
func listenersEnv(listeners []net.Listener) string {
	addrs := make([]string, 0)
	for _, ln := range listeners {
		addrs = append(addrs, ln.Addr().String())
	}
	return fmt.Sprintf("%s=%s", envKey, strings.Join(addrs, envSep))
}

func createListenerFiles(listeners []net.Listener) ([]*os.File, error) {
	fs := make([]*os.File, 0)
	var err error
loop:
	for _, l := range listeners {
		switch l := l.(type) {
		case *net.TCPListener:
			f, e := l.File()
			if e != nil {
				err = fmt.Errorf("graceful: failed to create listener file: %v", e)
				break loop
			}
			fs = append(fs, f)
		default:
			err = fmt.Errorf("graceful: failed to create listener file. not implemented %T", l)
			break loop
		}
	}
	if err != nil {
		for _, f := range fs {
			if err := f.Close(); err != nil {
				log.Printf("graceful: an error occurred %v", err)
			}
		}
		return nil, err
	}
	return fs, nil
}

func closeListenerFiles(files []*os.File) {
	for _, f := range files {
		if err := f.Close(); err != nil {
			log.Printf("graceful: failed to close listener files: %v", err)
		}
	}
}
