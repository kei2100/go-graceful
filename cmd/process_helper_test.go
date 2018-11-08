package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/go-ps"
)

type process struct {
	ps.Process
	children []*process
}

func (p *process) waitStartChildren(timeout time.Duration) error {
	ctx, can := context.WithTimeout(context.Background(), timeout)

	defer can()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to start child process: %v", ctx.Err())
		case <-tick.C:
			pp, err := findProcess(p.Pid())
			if err != nil {
				return fmt.Errorf("failed to find child process: %v", err)
			}
			if len(pp.children) == 0 {
				continue
			}
			p.children = pp.children
			return nil
		}
	}
}

func (p *process) childrenPids() []int {
	pids := make([]int, 0)
	for _, c := range p.children {
		pids = append(pids, c.Pid())
	}
	return pids
}

// findProcess looks up a single process by pid.
//
// process will be nil and error will be nil if a matching process is
// not found.
func findProcess(pid int) (*process, error) {
	prs, err := findProcesses(pid)
	if err != nil {
		return nil, err
	}
	if len(prs) != 1 {
		return nil, fmt.Errorf("pid %v not found", pid)
	}
	return prs[0], nil
}

// findProcesses looks up processes by pids.
// if all processes do not match pids, returns nil, nil
func findProcesses(pids ...int) ([]*process, error) {
	prs, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %v", err)
	}
	prByPid := make(map[int]ps.Process)     // process by pid map
	prsByPpid := make(map[int][]ps.Process) // processes by ppid map
	for _, pr := range prs {
		prByPid[pr.Pid()] = pr
		prsByPpid[pr.PPid()] = append(prsByPpid[pr.PPid()], pr)
	}

	var procs []*process
	for _, pid := range pids {
		if pr, ok := prByPid[pid]; ok { // match given pid
			var children []*process
			if cprs, ok := prsByPpid[pr.Pid()]; ok { // find children
				for _, cpr := range cprs {
					children = append(children, &process{Process: cpr})
				}
			}
			procs = append(procs, &process{Process: pr, children: children})
		}
	}
	return procs, nil
}

func waitNoProcess(timeout time.Duration, pids ...int) error {
	ctx, can := context.WithTimeout(context.Background(), timeout)
	defer can()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout exceeded while waiting for no process: %v", ctx.Err())
		case <-tick.C:
			prs, err := findProcesses(pids...)
			if err != nil {
				return fmt.Errorf("failed to find process: %v", err)
			}
			if len(prs) == 0 {
				return nil
			}
		}
	}
}
