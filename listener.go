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
	for i, addr := range InheritedAddrs() {
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

// InheritedAddrs lists inherited addrs from supervisor process.
// this func only for worker process
func InheritedAddrs() []string {
	return strings.Split(os.Getenv(envKey), envSep)
}

// InheritOrListenTCP returns inherited listener.
// if the addr is not included inherited addrs, create a new listener
func InheritOrListenTCP(addr string) (net.Listener, error) {
	for i, iaddr := range InheritedAddrs() {
		if iaddr != addr {
			continue
		}
		return func() (net.Listener, error) {
			fd := uintptr(3 + i)
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
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("graceful: failed to listen: %v", err)
	}
	return ln, nil
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
	for _, l := range listeners {
		switch l := l.(type) {
		case *net.TCPListener:
			f, e := l.File()
			if e != nil {
				closeListenerFiles(fs)
				return nil, fmt.Errorf("graceful: failed to create listener file: %v", e)
			}
			fs = append(fs, f)
		default:
			closeListenerFiles(fs)
			return nil, fmt.Errorf("graceful: failed to create listener file. not implemented %T", l)
		}
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
