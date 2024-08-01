// Package main handles vm
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

func VmApp() error {
	const (
		jsonFile      = ".json"
		screenName    = "vfu-vm-%s"
		startCommand  = "start"
		statusCommand = "status"
		listCommand   = "list"
	)

	args := os.Args
	var cmd string
	var sub string
	switch len(args) {
	case 2:
	case 3:
		sub = args[2]
	default:
		return errors.New("invalid argument passed")
	}
	cfg := struct {
		Directory string
	}{}
	if err := ReadConfig("vm", &cfg); err != nil {
		return err
	}
	dir := filepath.Join(os.Getenv("HOME"), cfg.Directory)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var machines []string
	for _, f := range files {
		name := f.Name()
		if m, ok := strings.CutSuffix(name, jsonFile); ok {
			machines = append(machines, m)
		}
	}
	cmd = args[1]
	switch cmd {
	case listCommand:
		for _, item := range machines {
			fmt.Println(item)
		}
		return nil
	case completionKeyword:
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		data := struct {
			Exe     string
			List    string
			Options string
			Start   string
		}{Start: startCommand, Exe: filepath.Base(exe), List: listCommand, Options: strings.Join([]string{listCommand, statusCommand, startCommand}, " ")}
		t, err := template.New("t").Parse(`#!/usr/bin/env bash

_{{ $.Exe }}() {
  local cur opts
  cur=${COMP_WORDS[COMP_CWORD]}
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "{{ $.Options }}" -- "$cur") )
  else
    if [ "$COMP_CWORD" -eq 2 ]; then
      case "${COMP_WORDS[1]}" in
        "{{ $.Start }}")
          COMPREPLY=( $(compgen -W "$({{ $.Exe }} {{ $.List }})" -- "$cur") )
          ;;
      esac
    fi
  fi
}

complete -F _{{ $.Exe }} -o bashdefault {{ $.Exe }}`)
		if err != nil {
			return err
		}
		return t.Execute(os.Stdout, data)
	case startCommand:
		if sub == "" {
			return errors.New("start requires machine")
		}
		if !slices.Contains(machines, sub) {
			return fmt.Errorf("unknown machine: %s", sub)
		}
		return exec.Command("screen", "-d", "-m", "-S", fmt.Sprintf(screenName, sub), "vfu", "--config", filepath.Join(dir, fmt.Sprintf("%s%s", sub, jsonFile))).Run()
	case statusCommand:
		printTable("vm", "status")
		fmt.Println("------------------")
		screens, _ := exec.Command("screen", "-list").CombinedOutput()
		s := string(screens)
		for _, machine := range machines {
			state := "stopped"
			screen := fmt.Sprintf(screenName, machine)
			if strings.Contains(s, screen) {
				state = "running"
			}
			printTable(machine, state)
		}
		return nil
	}
	return errors.New("invalid command")
}

func printTable(name, state string) {
	fmt.Printf("%-10s %s\n", name, state)
}
