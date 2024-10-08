package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"text/template"
)

func doBuildStep(data any, commands []string) error {
	if len(commands) == 0 {
		return nil
	}
	cmd := commands[0]
	var args []string
	for idx, c := range commands {
		if idx == 0 {
			continue
		}
		t, err := template.New("t").Parse(c)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return err
		}
		args = append(args, buf.String())
	}
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// BuildFromApp handles building from a source package of an app
func BuildFromApp(a Args) error {
	type build struct {
		Configure []string
		Build     []string
		Install   []string
	}
	cfg := Configuration[struct {
		Builds map[string]build
		Root   string
	}]{}
	if err := cfg.Load(a); err != nil {
		return err
	}
	found := false
	var rules build
	for k, v := range cfg.Settings.Builds {
		if PathExists(k) {
			rules = v
			found = true
			break
		}
	}
	if !found {
		return errors.New("unable to detect build type")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	data := struct {
		RootDir string
		CurDir  string
	}{cfg.Settings.Root, cwd}
	for _, set := range [][]string{rules.Configure, rules.Build, rules.Install} {
		if err := doBuildStep(data, set); err != nil {
			return err
		}
	}
	return nil
}
