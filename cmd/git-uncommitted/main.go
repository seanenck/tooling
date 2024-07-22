package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to read commit state: %v\n", err)
		os.Exit(1)
	}
}

func uncommit(stdout chan string, dir string) {
	cmd := exec.Command("git", "current-state")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err == nil {
		res := strings.TrimSpace(string(out))
		if res != "" {
			stdout <- res
			return
		}
	}
	stdout <- ""
}

func run() error {
	mode := flag.String("mode", "", "operating mode")
	flag.Parse()
	op := *mode
	if op == "pwd" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		state, _ := exec.Command("git", "-C", wd, "rev-parse", "--is-inside-work-tree").Output()
		if strings.TrimSpace(string(state)) == "true" {
			cmd := exec.Command("git", "current-state", "--quick")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		return nil
	}
	home := fmt.Sprintf("%s%c", os.Getenv("HOME"), os.PathSeparator)
	dirs, err := os.ReadFile(filepath.Join(home, ".config", "etc", "uncommit"))
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var all []chan string
	for _, l := range strings.Split(string(dirs), "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			path := filepath.Join(home, d)
			children, err := os.ReadDir(path)
			if err != nil {
				return
			}
			for _, child := range children {
				r := make(chan string)
				go func(dir string, out chan string) {
					uncommit(out, dir)
				}(filepath.Join(path, child.Name()), r)
				all = append(all, r)
			}
		}(trimmed)
	}
	wg.Wait()
	var results []string
	prefix := ""
	isMessage := op == "motd"
	if isMessage {
		prefix = "  "
	}
	for _, a := range all {
		res := <-a
		if res != "" {
			results = append(results, fmt.Sprintf("%s%s", prefix, strings.ReplaceAll(res, home, "")))
		}
	}
	if len(results) > 0 {
		if isMessage {
			fmt.Println("uncommitted\n===")
		}
		sort.Strings(results)
		fmt.Println(strings.Join(results, "\n"))
	}
	return nil
}
