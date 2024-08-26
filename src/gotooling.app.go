package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Gotooling handles go-based tool installs
func GotoolingApp(a Args) error {
	cfg := struct {
		Tools []string
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}
	for _, t := range cfg.Tools {
		fmt.Printf("go tool: %s\n", t)
		cmd := exec.Command("go", "install", t)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
