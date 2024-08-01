// Package main handles current state for a git repo
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	gitStatusCommand = "status"
	gitBranchCommand = "branch"
)

type (
	gitStatus struct {
		cmd string
		ok  bool
		err error
		dir gitPath
	}
	gitPath string
)

func (r gitStatus) write() {
	fmt.Printf("-> %s (%s)\n", r.dir, r.cmd)
}

func gitCommand(sub string, p gitPath, args ...string) gitStatus {
	resulting := gitStatus{cmd: sub, dir: p}
	arguments := []string{sub}
	arguments = append(arguments, args...)
	cmd := exec.Command("git", arguments...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err == nil {
		trimmed := strings.TrimSpace(string(out))
		switch sub {
		case gitStatusCommand:
			resulting.ok = !strings.Contains(trimmed, "[ahead")
		case gitBranchCommand:
			resulting.ok = strings.Contains(trimmed, "main") || strings.Contains(trimmed, "master")
		default:
			resulting.ok = trimmed == ""
		}
	} else {
		resulting.err = err
	}
	return resulting
}

// GitCurrentStateApp handles reporting state of git status for current directory
func GitCurrentStateApp() error {
	quick := flag.Bool("quick", false, "quickly exit on first issue")
	flag.Parse()
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	color := func(text string, mode int) {
		fmt.Printf("\x1b[%dm(%s)\x1b[0m", mode, text)
	}
	dirty := func() {
		color("dirty", 31)
	}
	directory := gitPath(dir)
	if directory == "" {
		return errors.New("directory must be set")
	}
	isQuick := *quick
	r := gitCommand("update-index", directory, "-q", "--refresh")
	if r.err != nil {
		return r.err
	}
	if !r.ok {
		if isQuick {
			dirty()
			return nil
		}
		r.write()
	}
	gitCommandAsync := func(res chan gitStatus, sub string, p gitPath, args ...string) {
		res <- gitCommand(sub, p, args...)
	}
	var results []chan gitStatus
	for sub, cmd := range map[string][]string{
		"diff-index":     {"--name-only", "HEAD", "--"},
		gitStatusCommand: {"-sb"},
		"ls-files":       {"--other", "--exclude-standard"},
		gitBranchCommand: {"--show-current"},
	} {
		r := make(chan gitStatus)
		go gitCommandAsync(r, sub, directory, cmd...)
		results = append(results, r)
	}

	done := false
	for _, r := range results {
		read := <-r
		if read.err != nil {
			continue
		}
		if !read.ok {
			if isQuick && !done {
				dirty()
				done = true
			}
			if !isQuick {
				read.write()
			}
		}
	}
	if done {
		return nil
	}

	if isQuick {
		color("clean", 32)
	}
	return nil
}
