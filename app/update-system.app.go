// Package main handles system update commands
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func UpdateSystemApp() error {
	cfg := struct {
		Updates []struct {
			Detect  string
			Command []string
		}
	}{}
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
