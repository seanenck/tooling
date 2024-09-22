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
	cfg := struct {
		Library    string
		URL        string
		Caffeinate bool
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}
	home := os.Getenv("HOME")
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
		opts := strings.Join(opt, " ")
		fxn := func(exe string) any {
			data := struct {
				Options string
				Exe     string
			}{Options: opts, Exe: exe}
			return data
		}
		const (
			zshCompletion = `#compdef _{{ $.Exe }} {{ $.Exe }}
_{{ $.Exe }}() {
  local curcontext="$curcontext" state
  typeset -A opt_args

  _arguments \
    '1: :->main'\
    '*: :->args'

  case $state in
    main)
      _arguments '1:main:({{ $.Options }})'
    ;;
  esac
}

compdef _{{ $.Exe }} {{ $.Exe }}
`
			bashCompletion = `#!/usr/bin/env bash

_{{ $.Exe }}() {
  local cur
  cur=${COMP_WORDS[COMP_CWORD]}
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "{{ $.Options }}" -- "$cur") )
  fi
}

complete -F _{{ $.Exe }} -o bashdefault {{ $.Exe }}`
		)
		return CompletionType{Bash: bashCompletion, Zsh: zshCompletion}.Generate(fxn)
	}
	if !slices.Contains(opt, cmd) {
		return fmt.Errorf("%s is an invalid library command", cmd)
	}
	if cfg.URL != "" {
		res, err := http.DefaultClient.Get(cfg.URL)
		if err != nil {
			return err
		}
		defer res.Body.Close()
	}
	exe := "caffeinate"
	var arguments []string
	script := filepath.Join(lib, cmd)
	if cfg.Caffeinate {
		arguments = append(arguments, script)
	} else {
		exe = script
		arguments = append(arguments, sub...)
	}
	c := exec.Command(exe, arguments...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
