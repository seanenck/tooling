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
	statusCommand = "status"
	branchCommand = "branch"
)

type (
	result struct {
		cmd string
		ok  bool
		err error
		dir path
	}
	path string
)

func (r result) write() {
	fmt.Printf("-> %s (%s)\n", r.dir, r.cmd)
}

func gitCommand(sub string, p path, args ...string) result {
	resulting := result{cmd: sub, dir: p}
	arguments := []string{sub}
	arguments = append(arguments, args...)
	cmd := exec.Command("git", arguments...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err == nil {
		trimmed := strings.TrimSpace(string(out))
		switch sub {
		case statusCommand:
			resulting.ok = !strings.Contains(trimmed, "[ahead")
		case branchCommand:
			resulting.ok = strings.Contains(trimmed, "main") || strings.Contains(trimmed, "master")
		default:
			resulting.ok = trimmed == ""
		}
	} else {
		resulting.err = err
	}
	return resulting
}

func gitCommandAsync(res chan result, sub string, p path, args ...string) {
	res <- gitCommand(sub, p, args...)
}

func color(text string, mode int) {
	fmt.Printf("\x1b[%dm(%s)\x1b[0m", mode, text)
}

func dirty() {
	color("dirty", 31)
}

func run() error {
	quick := flag.Bool("quick", false, "quickly exit on first issue")
	flag.Parse()
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	directory := path(dir)
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
	var results []chan result
	for sub, cmd := range map[string][]string{
		"diff-index":  {"--name-only", "HEAD", "--"},
		statusCommand: {"-sb"},
		"ls-files":    {"--other", "--exclude-standard"},
		branchCommand: {"--show-current"},
	} {
		r := make(chan result)
		go gitCommandAsync(r, sub, directory, cmd...)
		results = append(results, r)
	}

	for _, r := range results {
		read := <-r
		if read.err != nil {
			continue
		}
		if !read.ok {
			if isQuick {
				dirty()
				return nil
			}
			read.write()
		}
	}

	if isQuick {
		color("clean", 32)
	}
	return nil
}
