// Package main handles hooks to push to github
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// GithubHooksApp generates hooks for auto-pushing
func GithubHooksApp() error {
	home := os.Getenv("HOME")
	cfg := struct {
		Path            string
		User            string
		PasswordCommand []string
	}{}
	if err := ReadConfig("hooks", &cfg); err != nil {
		return err
	}
	path := filepath.Join(home, cfg.Path)
	dirs, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	t, err := template.New("t").Parse(`#!/bin/sh
for OBJECT in all tags; do
    if ! git -C "{{ $.Path }}" push --$OBJECT "https://{{ $.Token }}@github.com/seanenck/{{ $.Name }}"; then
        echo
        echo "================="
        echo "failed to push $OBJECT for {{ $.Name }}"
        echo "================="
        echo
        exit 1
    fi
done`)
	if err != nil {
		return err
	}
	pass := cfg.PasswordCommand
	cmd := pass[0]
	var args []string
	if len(pass) > 1 {
		args = pass[1:]
	}
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return err
	}
	token := fmt.Sprintf("%s:%s", cfg.User, strings.TrimSpace(string(out)))
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		obj := struct {
			Path  string
			Name  string
			Token string
		}{Name: d.Name()}
		obj.Path = filepath.Join(path, obj.Name)
		obj.Token = token
		hook := filepath.Join(obj.Path, "hooks", "post-receive")
		var buf bytes.Buffer
		if err := t.Execute(&buf, obj); err != nil {
			return err
		}
		if err := os.WriteFile(hook, buf.Bytes(), 0o755); err != nil {
			return err
		}
	}
	return nil
}
