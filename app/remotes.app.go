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
func RemotesApp() error {
	const (
		gitMode  = "Git"
		brewMode = "Brew"
	)
	home := os.Getenv("HOME")
	cfg := struct {
		Sources map[string]string
		State   string
		Modes   map[string]map[string]interface{}
	}{}
	if err := ReadConfig(&cfg); err != nil {
		return err
	}
	var filters []*regexp.Regexp
	modes := cfg.Modes
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
	brewRawField, err := parseModeOpts[string](brewMode, "Field", modes)
	if err != nil {
		return err
	}
	brewField := strings.Split(*brewRawField, ".")
	brewLength := len(brewField) - 1
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
	for source, typed := range cfg.Sources {
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
		case brewMode:
			var data []map[string]interface{}
			if err := json.Unmarshal(out, &data); err != nil {
				return err
			}
			for _, d := range data {
				use := d
				for idx, p := range brewField {
					vals, ok := use[p]
					if !ok {
						return fmt.Errorf("%s is missing required field: %v", source, brewField)
					}
					if idx == brewLength {
						s, ok := vals.(string)
						if !ok {
							return fmt.Errorf("%s field is not string: %v", source, vals)
						}
						versioner(source, s)
					} else {
						next, ok := vals.(map[string]interface{})
						if !ok {
							return fmt.Errorf("subsection is invalid: %s (%v)", source, brewField)
						}
						use = next
					}
				}
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
