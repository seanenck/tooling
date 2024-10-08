// Package main handles various git uncommitted states for output
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

// GitUncommittedApp handles a summary of repositories across a set of directories
func GitUncommittedApp(a Args) error {
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
	cfg := Configuration[struct {
		Directories []string
	}]{}
	if err := cfg.Load(a); err != nil {
		return err
	}

	var wg sync.WaitGroup
	var all []chan string
	home := fmt.Sprintf("%s%c", os.Getenv("HOME"), os.PathSeparator)
	for _, dir := range cfg.Settings.Directories {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			path := filepath.Join(home, d)
			children, err := os.ReadDir(path)
			if err != nil {
				return
			}
			for _, child := range children {
				childPath := filepath.Join(path, child.Name())
				if !PathExists(filepath.Join(childPath, ".git")) {
					continue
				}
				r := make(chan string)
				go func(dir string, out chan string) {
					uncommit(out, dir)
				}(childPath, r)
				all = append(all, r)
			}
		}(dir)
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
			for _, line := range strings.Split(res, "\n") {
				results = append(results, fmt.Sprintf("%s%s", prefix, strings.Replace(line, home, "", 1)))
			}
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
