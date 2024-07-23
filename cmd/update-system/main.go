// Package main handles system update commands
package main

import (
	"fmt"
	"os"
	"os/exec"
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
	var cfg Config
	if err := ReadConfig("updates", &cfg); err != nil {
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
