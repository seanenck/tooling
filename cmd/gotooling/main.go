// Package main handles go tooling installs/updates
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Config handles tool configuration
type Config struct {
	Tools []string
}

func run() error {
	read, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "etc", "gotools"))
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(read, &cfg); err != nil {
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
