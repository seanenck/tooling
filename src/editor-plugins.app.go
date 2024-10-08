// Package main provides editor plugin help
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type (
	// Plugin is a specific editor plugin remote
	Plugin string
)

func (p Plugin) write(text string) {
	fmt.Printf("  -> %s: %s\n", p, text)
}

func (p Plugin) fail() {
	p.write("fail")
}

func updatePlugin(dest, plugin string) {
	base := Plugin(filepath.Base(plugin))
	to := filepath.Join(dest, string(base))
	base.write("sync")
	var args []string
	if PathExists(to) {
		b, err := exec.Command("git", "-C", to, "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			base.fail()
			return
		}
		args = []string{"-C", to, "pull", "--quiet", "origin", strings.TrimSpace(string(b))}
	} else {
		args = []string{"clone", "--quiet", plugin, to, "--single-branch"}
	}
	git := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	if err := git(args...); err != nil {
		base.fail()
		return
	}
	base.write("done")
}

// EditorPluginsApp handles getting/updating editor plugins
func EditorPluginsApp(a Args) error {
	home := os.Getenv("HOME")
	type config []struct {
		Path    string
		Enabled bool
		Plugins []string
	}
	cfg := Configuration[struct{ Plugins config }]{}
	if err := cfg.Load(a); err != nil {
		return err
	}

	for _, c := range cfg.Settings.Plugins {
		if !c.Enabled {
			continue
		}
		fmt.Printf("updating: %s\n", c.Path)
		dest := filepath.Join(home, c.Path)
		var wg sync.WaitGroup
		for _, plugin := range c.Plugins {
			wg.Add(1)
			go func(to, remote string) {
				defer wg.Done()
				updatePlugin(to, remote)
			}(dest, plugin)
		}

		wg.Wait()
	}
	return nil
}
