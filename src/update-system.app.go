// Package main handles system update commands
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"
)

// UpdateSystemApp handles system update calls
func UpdateSystemApp(a Args) error {
	cfg := struct {
		Path   string
		Format string
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
	state := filepath.Join(os.Getenv("HOME"), cfg.Path)
	if !force {
		if PathExists(state) {
			i, err := os.Stat(state)
			if err != nil {
				return err
			}
			now := time.Now().Format(cfg.Format)
			if i.ModTime().Format(cfg.Format) == now {
				fmt.Println("up-to-date")
				return nil
			}
		}
	}
	var updates []string
	for k, v := range a.Flags {
		if slices.Contains(v, a.Name) {
			updates = append(updates, k)
		}
	}
	for _, cmd := range updates {
		fmt.Printf("updating: %v\n", cmd)
		c := exec.Command(cmd)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return err
		}
	}
	return os.WriteFile(state, []byte{}, 0o644)
}
