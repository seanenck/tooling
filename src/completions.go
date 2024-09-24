package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const (
	// CompletionKeyword is the common completion keyword for bash completions
	CompletionKeyword = "completions"
)

// CompletionType help setup completion templating
type CompletionType struct {
	Bash       string
	Zsh        string
	ZshCompDef bool
}

// Generate will generate a multi-shell completion
func (c CompletionType) Generate(data any) error {
	shell := os.Getenv("SHELL")
	text := ""
	switch shell {
	case "/bin/bash":
		text = c.Bash
	case "/bin/zsh":
		if c.ZshCompDef {
			text = fmt.Sprintf("#compdef _{{ $.Exe }}\n\n_{{ $.Exe }}() {\n%s\n}\n\ncompdef _{{ $.Exe }} {{ $.Exe }}", strings.TrimPrefix(c.Zsh, "\n"))
		} else {
			text = c.Zsh
		}
	default:
		return fmt.Errorf("no completions for: %s", shell)
	}
	if text == "" {
		return fmt.Errorf("empty completion: %s", shell)
	}
	t, err := template.New("t").Parse(text)
	if err != nil {
		return err
	}
	return t.Execute(os.Stdout, data)
}
