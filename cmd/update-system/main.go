// Package main handles system update commands
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	// Config handles tool configuration
	Config struct {
		Updates []Command
	}

	// Command defines how to detect a update system and run it
	Command struct {
		Detect  string
		Command []string
	}
)

func run() error {
	read, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "etc", "updates"))
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(read, &cfg); err != nil {
		return err
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
	return nil
}
