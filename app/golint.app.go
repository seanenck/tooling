// Package main handles go linting helpers
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// GolintApp handles golint wrapping of tools
func GolintApp(a Args) error {
	if !PathExists("go.mod") {
		return errors.New("cowardly refusing to run outside go.mod root")
	}
	type (
		Tool struct {
			Name    string
			Detect  bool
			Command []string
		}
	)
	cfg := struct {
		Tools []Tool
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}
	searched := false
	var files []string
	var length int
	var commands []Tool
	for _, tool := range cfg.Tools {
		if l := len(tool.Name); l > length {
			length = l
		}
		args := tool.Command
		if tool.Detect {
			if !searched {
				err := filepath.Walk(".", func(path string, _ fs.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.HasSuffix(path, ".go") {
						files = append(files, path)
					}
					return nil
				})
				if err != nil {
					return err
				}
				searched = true
			}
			args = append(args, files...)
		} else {
			args = append(args, "./...")
		}
		commands = append(commands, Tool{Name: tool.Name, Command: args})
	}
	formatter := "%-" + fmt.Sprintf("%d", length) + "s: %s\n"
	for _, command := range commands {
		if len(command.Command) < 2 {
			return errors.New("invalid definition for command")
		}
		exe := command.Command[0]
		args := command.Command[1:]
		out, _ := exec.Command(exe, args...).CombinedOutput()
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			fmt.Printf(formatter, command.Name, t)
		}
	}
	return nil
}
