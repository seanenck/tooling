// Package main handles system update commands
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// UpdateSystemApp handles system update calls
func UpdateSystemApp(a Args) error {
	cfg := struct {
		State struct {
			Path   string
			Format string
		}
		Updates []struct {
			Detect  string
			Command []string
		}
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}
	args := os.Args
	force := false
	switch len(args) {
	case 1:
		break
	case 2:
		switch args[1] {
		case "--force":
			force = true
		default:
			return fmt.Errorf("unknown argument: %v", args)
		}
	default:
		return fmt.Errorf("unknown arguments given: %v", args)
	}
	state := filepath.Join(os.Getenv("HOME"), cfg.State.Path)
	if !force {
		if PathExists(state) {
			i, err := os.Stat(state)
			if err != nil {
				return err
			}
			now := time.Now().Format(cfg.State.Format)
			if i.ModTime().Format(cfg.State.Format) == now {
				fmt.Println("up-to-date")
				return nil
			}
		}
	}
	for _, cmd := range cfg.Updates {
		out, err := exec.Command("command", "-v", cmd.Detect).Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			fmt.Printf("updating: %v\n", cmd.Command)
			command := cmd.Command[0]
			var args []string
			if len(cmd.Command) > 1 {
				args = cmd.Command[1:]
			}
			c := exec.Command(command, args...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return err
			}
		}
	}
	return os.WriteFile(state, []byte{}, 0o644)
}
