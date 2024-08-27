// Package main helps manage data
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

// ManageDataApp handles management of data (wrappers)
func ManageDataApp(a Args) error {
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
		URL     string
		Remote  bool
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
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
	res, err := http.DefaultClient.Get(cfg.URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	exe := "caffeinate"
	var arguments []string
	script := filepath.Join(lib, cmd)
	if cfg.Remote {
		const (
			sshFlag = "--ssh"
			sshEnv  = "IS_SSH_TASKS"
		)
		exe = script
		arguments = sub
		if !slices.Contains(sub, sshFlag) && os.Getenv(sshEnv) == "" {
			return errors.New("unable to work in remote mode without ssh flag/env")
		}
		os.Setenv(sshEnv, "true")
		arguments = func() []string {
			var r []string
			for _, f := range sub {
				if f == sshFlag {
					continue
				}
				r = append(r, f)
			}
			return r
		}()
	} else {
		arguments = append(arguments, script)
		arguments = append(arguments, sub...)
	}
	c := exec.Command(exe, arguments...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
