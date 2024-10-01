// Package main helps manage remote version information
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// RemotesApp helps sync release tags from remotes for update tracking
func RemotesApp(a Args) error {
	home := os.Getenv("HOME")
	type modeType struct {
		Command   string
		Arguments []string
		Filter    string
	}
	cfg := struct {
		Sources map[string]string
		State   string
		Modes   map[string]modeType
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}

	state := filepath.Join(home, cfg.State)
	var had []string
	isInit := !PathExists(state)
	if isInit {
		fmt.Println("initializing...")
	} else {
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
	}
	var now []string
	versioner := func(n, v string) {
		now = append(now, fmt.Sprintf("%s %s", n, v))
	}
	filterSet := make(map[string]*regexp.Regexp)
	for k, v := range cfg.Modes {
		r, err := regexp.Compile(v.Filter)
		if err != nil {
			return err
		}
		filterSet[k] = r
	}
	for source, typed := range cfg.Sources {
		cmd, ok := cfg.Modes[typed]
		if !ok {
			return fmt.Errorf("unknown source mode type: %s (%s)", typed, source)
		}
		fmt.Printf("getting: %s\n", source)
		exe := cmd.Command
		args := cmd.Arguments
		args = append(args, source)
		out, err := exec.Command(exe, args...).Output()
		if err != nil {
			return err
		}
		if len(cmd.Filter) == 0 {
			return fmt.Errorf("no filters for: %s", source)
		}
		name := filepath.Base(source)
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			filter, ok := filterSet[typed]
			if !ok {
				return fmt.Errorf("no filters for type: %s", typed)
			}
			matches := filter.FindStringSubmatch(line)
			if len(matches) > 0 {
				versioner(name, matches[1])
			}
		}
	}

	sort.Strings(now)
	if len(had) > 0 || isInit {
		differ := func(prefix rune, left, right []string) bool {
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

		older := differ('-', had, now)
		newer := differ('+', now, had)
		if older || newer || isInit {
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
