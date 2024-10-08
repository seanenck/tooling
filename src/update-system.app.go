// Package main handles system update commands
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// UpdateSystemApp handles system update calls
func UpdateSystemApp(a Args) error {
	cfg := Configuration[struct {
		Path   string
		Format string
	}]{}
	if err := cfg.Load(a); err != nil {
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
	state := filepath.Join(os.Getenv("HOME"), cfg.Settings.Path)
	if !force {
		if PathExists(state) {
			i, err := os.Stat(state)
			if err != nil {
				return err
			}
			now := time.Now().Format(cfg.Settings.Format)
			if i.ModTime().Format(cfg.Settings.Format) == now {
				fmt.Println("up-to-date")
				return nil
			}
		}
	}
	var updates []string
	files, err := os.ReadDir(a.Config.Dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, a.Config.Extension) {
			c := Configuration[struct{}]{}
			if err := c.LoadFile(filepath.Join(a.Config.Dir, name)); err != nil {
				return err
			}
			if slices.Contains(c.Flags, a.Name) {
				updates = append(updates, strings.TrimSuffix(name, a.Config.Extension))
			}
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
