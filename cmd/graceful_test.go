package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path"
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

	if err := g.stopGraceful(time.Second); err != nil {
		t.Fatal(err)
	}
	if err := waitNoProcess(time.Second, append(process.childrenPids(), process.Pid())...); err != nil {
		t.Fatal(err)
	}
}
