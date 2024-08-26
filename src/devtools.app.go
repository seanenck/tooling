package main

import (
	"fmt"
	"os"
	"os/exec"
)

func updateByTool(tool string, args, remotes []string) error {
	for _, t := range remotes {
		fmt.Printf("  -> %s\n", t)
		var a []string
		a = append(a, args...)
		a = append(a, t)
		cmd := exec.Command(tool, a...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// DevtoolsApp helps manage developer tool installs
func DevtoolsApp(a Args) error {
	cfg := struct {
		Go    []string
		Cargo []string
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}

	for k, v := range map[string][]string{
		"go":    cfg.Go,
		"cargo": cfg.Cargo,
	} {
		fmt.Printf("%s updates:\n", k)
		if err := updateByTool(k, []string{"install"}, v); err != nil {
			return err
		}
	}

	return nil
}
