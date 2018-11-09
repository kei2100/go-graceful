package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path"
	"sync"
	"testing"
	"time"
)

func init() {
	goFiles := []string{
		path.Join("testdata", "stub_http.go"),
		path.Join("graceful.go"),
	}
	for _, f := range goFiles {
		cmd := exec.Command("go", "build", f)
		if err := cmd.Start(); err != nil {
			log.Panicf("faield to build %s: %v", f, err)
		}
		if err := cmd.Wait(); err != nil {
			log.Panicf("faield to build %s: %v", f, err)
		}
	}
}

func TestGraceful_Start_Shutdown(t *testing.T) {
	g, err := startGraceful()
	if err != nil {
		t.Fatal(err)
	}
	process, err := findProcess(g.cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := process.waitStartChildren(time.Second); err != nil {
		t.Fatal(err)
	}

	testGet(t, fmt.Sprintf("http://%s/ping", g.listenAddr))

	if err := g.stopGraceful(time.Second); err != nil {
		t.Fatal(err)
	}
	if err := waitNoProcess(time.Second, append(process.childrenPids(), process.Pid())...); err != nil {
		t.Fatal(err)
	}
}

func TestGraceful_Restart(t *testing.T) {
	g, err := startGraceful()
	if err != nil {
		t.Fatal(err)
	}
	process, err := findProcess(g.cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := process.waitStartChildren(time.Second); err != nil {
		t.Fatal(err)
	}

	go testGet(t, fmt.Sprintf("http://%s/delay", g.listenAddr))
	time.Sleep(100 * time.Millisecond)

	if err := g.restartGraceful(); err != nil {
		t.Fatal(err)
	}
	if err := waitNoProcess(10*time.Second, process.childrenPids()...); err != nil {
		t.Fatal(err)
	}
	if err := process.waitStartChildren(time.Second); err != nil {
		t.Fatal(err)
	}

	testGet(t, fmt.Sprintf("http://%s/delay", g.listenAddr))

	if err := g.stopGraceful(3*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := waitNoProcess(time.Second, append(process.childrenPids(), process.Pid())...); err != nil {
		t.Fatal(err)
	}
}

func TestGraceful_Restart_Multi(t *testing.T) {
	g, err := startGraceful()
	if err != nil {
		t.Fatal(err)
	}
	process, err := findProcess(g.cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := process.waitStartChildren(time.Second); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	requester := func() {
		defer wg.Done()
		tick := time.NewTicker(100*time.Millisecond)
		defer tick.Stop()
		for   {
			select {
			case <- tick.C:
				go testGet(t, fmt.Sprintf("http://%s/delay", g.listenAddr))
			case <-stop:
				return
			}
		}
	}

	restarter := func() {
		defer wg.Done()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <- tick.C:
				g.restartGraceful()
			case <-stop:
				return
			}
		}
	}

	wg.Add(2)
	go requester()
	go restarter()
	time.Sleep(15*time.Second)
	close(stop)
	wg.Wait()

	if err := process.waitStartChildren(time.Second); err != nil {
		t.Fatal(err)
	}
	if err := g.stopGraceful(3*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := waitNoProcess(time.Second, append(process.childrenPids(), process.Pid())...); err != nil {
		t.Fatal(err)
	}
}

func testGet(t *testing.T, url string) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Errorf("response code got %v, want 200", res.StatusCode)
	}
}
