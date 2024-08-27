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

func parseModeOpts[T any](mode, key string, modes map[string]map[string]interface{}) (*T, error) {
	val, ok := modes[mode]
	if !ok {
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
	read, ok := val[key]
	if !ok {
		return nil, fmt.Errorf("unknown mode key: %s %s", mode, key)
	}
	actual, ok := read.(T)
	if !ok {
		return nil, fmt.Errorf("unknown mode key, is not of type: %s %s", mode, key)
	}
	return &actual, nil
}

// RemotesApp helps sync release tags from remotes for update tracking
func RemotesApp(a Args) error {
	const (
		gitMode = "Git"
		webMode = "HTTP"
	)
	home := os.Getenv("HOME")
	cfg := struct {
		Sources map[string]string
		State   string
		Modes   map[string]map[string]interface{}
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}

	modes := cfg.Modes
	rawWebModes, err := parseModeOpts[map[string]interface{}](webMode, "Filters", modes)
	if err != nil {
		return err
	}
	webModes := *rawWebModes

	var filters []*regexp.Regexp
	gitFilters, err := parseModeOpts[[]interface{}](gitMode, "Filter", modes)
	if err != nil {
		return err
	}
	for _, f := range *gitFilters {
		text, ok := f.(string)
		if !ok {
			return fmt.Errorf("unable to compile filter, unknown type: %v", f)
		}
		r, err := regexp.Compile(text)
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
	cmds := make(map[string][]string)
	for k, v := range modes {
		c, ok := v["Command"]
		if !ok {
			return fmt.Errorf("unable to find command for mode: %s", k)
		}
		raw, ok := c.([]interface{})
		if !ok {
			return fmt.Errorf("command for %s is not array", k)
		}
		var actions []string
		for _, r := range raw {
			s, ok := r.(string)
			if !ok {
				return fmt.Errorf("command for %s is not string: %v", k, raw)
			}
			actions = append(actions, s)
		}
		if len(actions) == 0 {
			return fmt.Errorf("empty commands: %s", k)
		}
		cmds[k] = actions
	}
	var now []string
	versioner := func(n, v string) {
		now = append(now, fmt.Sprintf("%s %s", n, v))
	}
	for source, raw := range cfg.Sources {
		typed := raw
		subType := ""
		if strings.Contains(raw, "|") {
			parts := strings.Split(raw, "|")
			if len(parts) != 2 {
				return fmt.Errorf("invalid sub type: %v", parts)
			}
			typed = parts[0]
			subType = parts[1]
		}
		fmt.Printf("getting: %s\n", source)
		cmd, ok := cmds[typed]
		if !ok {
			return fmt.Errorf("unknown remote mode: %s", typed)
		}
		exe := cmd[0]
		var args []string
		if len(cmd) > 1 {
			args = append(args, cmd[1:]...)
		}
		args = append(args, source)
		out, err := exec.Command(exe, args...).Output()
		if err != nil {
			return err
		}
		switch typed {
		case webMode:
			s, ok := webModes[subType]
			if !ok {
				return fmt.Errorf("unknown web mode: %s", subType)
			}
			def, ok := s.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid definition for web mode: %s %v", subType, def)
			}
			var start, end string
			for k, v := range def {
				r, ok := v.(string)
				if !ok {
					continue
				}
				switch k {
				case "Start":
					start = r
				case "End":
					end = r
				default:
					return fmt.Errorf("unexpected web selector key: %s", k)
				}
			}

			if start == "" || end == "" {
				return fmt.Errorf("invalid web selector: %v", def)
			}
			startRegex, err := regexp.Compile(start)
			if err != nil {
				return err
			}
			endRegex, err := regexp.Compile(end)
			if err != nil {
				return err
			}

			for _, line := range strings.Split(string(out), "\n") {
				from := startRegex.FindStringIndex(line)
				if len(from) == 0 {
					continue
				}
				selected := line[from[0]:]
				to := endRegex.FindStringIndex(selected)
				if len(to) == 0 {
					continue
				}
				selected = selected[0:to[1]]
				versioner(subType, selected)
			}
		case gitMode:
			{
				name := filepath.Base(source)
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
							versioner(name, part)
						}
					}
				}
			}
		}
	}

	sort.Strings(now)
	if len(had) > 0 {
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
