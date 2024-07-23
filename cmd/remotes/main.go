// Package main helps manage remote version information
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Config handles tool configuration
type Config struct {
	Remotes []string
	State   string
	Filter  []string
}

func differ(prefix rune, left, right []string) bool {
	status := false
	for _, item := range left {
		if slices.Contains(right, item) {
			continue
		}
		fmt.Printf("%c %s\n", prefix, item)
		status = true
	}
	return status
}

func run() error {
	home := os.Getenv("HOME")
	read, err := os.ReadFile(filepath.Join(home, ".config", "etc", "remotes"))
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(read, &cfg); err != nil {
		return err
	}
	var filters []*regexp.Regexp
	for _, f := range cfg.Filter {
		r, err := regexp.Compile(f)
		if err != nil {
			return err
		}
		filters = append(filters, r)
	}
	state := filepath.Join(home, cfg.State)
	var had []string
	if PathExists(state) {
		last, err := os.ReadFile(state)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(last)), "\n") {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			had = append(had, t)
		}
	} else {
		fmt.Println("initializing...")
	}
	var now []string
	for _, remote := range cfg.Remotes {
		fmt.Printf("getting: %s\n", remote)
		name := filepath.Base(remote)
		out, err := exec.Command("git", "ls-remote", "--tags", remote).Output()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) > 1 {
				part := strings.Join(parts[1:], " ")
				allowed := true
				for _, r := range filters {
					if r.MatchString(part) {
						continue
					}
					allowed = false
				}
				if allowed {
					now = append(now, fmt.Sprintf("%s %s", name, part))
				}
			}
		}
	}

	if len(had) > 0 {
		older := differ('-', had, now)
		newer := differ('+', now, had)
		if older || newer {
			fmt.Printf("updates applied? (y/N) ")
			reader := bufio.NewReader(os.Stdin)
			line, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			switch strings.ToLower(strings.TrimSpace(line)) {
			case "y":
				return os.WriteFile(state, []byte(strings.Join(now, "\n")), 0o644)
			}
		}
	}

	return nil
}
