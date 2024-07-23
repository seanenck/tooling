// Package main handles go tooling installs/updates
package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Config handles tool configuration
type Config struct {
	Tools []string
}

func run() error {
	var cfg Config
	if err := ReadConfig("gotools", &cfg); err != nil {
		return err
	}
	for _, tool := range cfg.Tools {
		fmt.Printf("installing: %s\n", tool)
		cmd := exec.Command("go", "install", tool)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
