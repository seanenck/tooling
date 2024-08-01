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

// ManageDataApp handles management of data (wrappers)
func ManageDataApp() error {
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
	cfg := struct {
		Library string
	}{}
	if err := ReadConfig(&cfg); err != nil {
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
	if cmd == CompletionKeyword {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		opts := strings.Join(opt, " ")
		data := struct {
			Options string
			Exe     string
		}{Options: opts, Exe: filepath.Base(exe)}
		t, err := template.New("t").Parse(`#!/usr/bin/env bash

_{{ $.Exe }}() {
  local cur
  cur=${COMP_WORDS[COMP_CWORD]}
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "{{ $.Options }}" -- "$cur") )
  fi
}

complete -F _{{ $.Exe }} -o bashdefault {{ $.Exe }}`)
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
