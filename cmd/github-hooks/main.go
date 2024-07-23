// Package main handles hooks to push to github
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const postReceive = `
#!/bin/sh
for OBJECT in all tags; do
    if ! git -C "{{ $.Path }}" push --$OBJECT "https://{{ $.Token }}@github.com/seanenck/{{ $.Name }}"; then
        echo
        echo "================="
        echo "failed to push $OBJECT for {{ $.Name }}"
        echo "================="
        echo
        exit 1
    fi
done
`

type (
	// Config handles tool configuration
	Config struct {
		Path            string
		User            string
		PasswordCommand []string
	}
	// Data handles templating output hook files
	Data struct {
		Path  string
		Name  string
		Token string
	}
)

func run() error {
	home := os.Getenv("HOME")
	read, err := os.ReadFile(filepath.Join(home, ".config", "etc", "hooks"))
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(read, &cfg); err != nil {
		return err
	}
	path := filepath.Join(home, cfg.Path)
	dirs, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	t, err := template.New("t").Parse(strings.TrimSpace(postReceive))
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
		obj := Data{Name: d.Name()}
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
