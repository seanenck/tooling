// Package main helps manage data
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

const completion = `#!/usr/bin/env bash

_manage-data() {
  local cur
  cur=${COMP_WORDS[COMP_CWORD]}
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "{{ $.Options }}" -- "$cur") )
  fi
}

complete -F _manage-data -o bashdefault manage-data`

// Config handles tool configuration
type Config struct {
	Library string
}

func run() error {
	args := os.Args
	if len(args) < 2 {
		return errors.New("invalid command")
	}
	cmd := args[1]
	var sub []string
	if len(args) > 1 {
		sub = args[2:]
	}
	home := os.Getenv("HOME")
	var cfg Config
	if err := ReadConfig("data", &cfg); err != nil {
		return err
	}
	lib := filepath.Join(home, cfg.Library)
	files, err := os.ReadDir(lib)
	if err != nil {
		return err
	}
	var opt []string
	for _, f := range files {
		opt = append(opt, f.Name())
	}
	if cmd == "completions" {
		opts := strings.Join(opt, " ")
		data := struct {
			Options string
		}{Options: opts}
		t, err := template.New("t").Parse(completion)
		if err != nil {
			return err
		}
		return t.Execute(os.Stdout, data)
	}
	if !slices.Contains(opt, cmd) {
		return fmt.Errorf("%s is an invalid library command", cmd)
	}
	arguments := []string{filepath.Join(lib, cmd)}
	arguments = append(arguments, sub...)
	c := exec.Command("caffeinate", arguments...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
