// Package main handles various utility needs
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

const (
	appDir   = "app"
	appFile  = ".app.go"
	buildDir = "target"
	tmpDir   = "tmp"
	mainText = `// Package main handles {{ .App }}
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := {{ .App }}(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}`
)

var (
	destDir    = filepath.Join(".local", "bin")
	buildFlags = []string{
		"-trimpath",
		"-buildmode=pie",
		"-mod=readonly",
		"-modcacherw",
		"-buildvcs=false",
	}
)

func main() {
	if err := build(); err != nil {
		fmt.Fprintf(os.Stderr, "\n===\nbuild failed: %v", err)
		os.Exit(1)
	}
}

func mkDirP(dir string) error {
	return runCommand("mkdir", "-p", dir)
}

func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func build() error {
	args := os.Args
	isInstall := false
	isClean := false
	switch len(args) {
	case 1:
	case 2:
		arg := args[1]
		switch arg {
		case "clean":
			isClean = true
		case "install":
			isInstall = true
		default:
			return fmt.Errorf("unknown argument: %s", arg)
		}
	default:
		return fmt.Errorf("unknown arguments: %v", args)
	}
	if err := mkDirP(buildDir); err != nil {
		return err
	}
	if isClean {
		return os.RemoveAll(buildDir)
	}
	if isInstall {
		dest := filepath.Join(os.Getenv("HOME"), destDir)
		files, err := os.ReadDir(buildDir)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			to := filepath.Join(dest, name)
			fmt.Printf("install -> %s (destination: %s)\n", name, to)
			if err := runCommand("install", "-m755", filepath.Join(buildDir, name), to); err != nil {
				return err
			}
		}
		return nil
	}
	files, err := os.ReadDir(appDir)
	if err != nil {
		return err
	}
	var source []string
	var targets []string
	maxName := 0
	for _, f := range files {
		name := f.Name()
		cut, ok := strings.CutSuffix(name, appFile)
		if ok {
			length := len(cut)
			if length > maxName {
				maxName = length
			}
			targets = append(targets, cut)
		} else {
			source = append(source, filepath.Join(appDir, name))
		}
	}
	tmpl, err := template.New("t").Parse(mainText)
	if err != nil {
		return err
	}
	formatter := fmt.Sprintf("%%s%%%ds -> ", maxName+5)
	for idx, target := range targets {
		prefix := "\n"
		if idx == 0 {
			prefix = ""
		}
		fmt.Printf(formatter, prefix, target)
		src := []string{filepath.Join(appDir, fmt.Sprintf("%s%s", target, appFile))}
		src = append(src, source...)
		obj := filepath.Join(buildDir, target)
		stat, err := os.Stat(obj)
		building := true
		if err == nil {
			building = false
			mod := stat.ModTime()
			checks := []string{"go.mod"}
			checks = append(checks, src...)
			for _, f := range checks {
				info, err := os.Stat(f)
				if err != nil {
					return err
				}
				if info.ModTime().After(mod) {
					building = true
					break
				}
			}
		}
		if !building {
			fmt.Printf("up-to-date")
			continue
		}

		tmp := filepath.Join(buildDir, tmpDir)
		os.RemoveAll(tmp)
		if err := mkDirP(tmp); err != nil {
			return err
		}
		isUpper := true
		properName := ""
		for _, r := range target {
			if (r >= 'a' && r <= 'z') || r == '-' {
				if r == '-' && !isUpper {
					isUpper = true
					continue
				}
				use := fmt.Sprintf("%c", r)
				if isUpper {
					use = strings.ToUpper(use)
					isUpper = false
				}
				properName = fmt.Sprintf("%s%s", properName, use)
			}
		}
		if properName == "" {
			return fmt.Errorf("unable to parse target proper name: %s", target)
		}
		properName = fmt.Sprintf("%sApp", properName)
		app := struct {
			App string
		}{properName}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, app); err != nil {
			return err
		}
		mainFile := filepath.Join(tmp, "main.go")
		if err := os.WriteFile(mainFile, buf.Bytes(), 0o644); err != nil {
			return err
		}
		inputs := []string{mainFile}
		for _, s := range src {
			name := filepath.Base(s)
			to := filepath.Join(tmp, name)
			if err := runCommand("cp", s, to); err != nil {
				return err
			}
			inputs = append(inputs, to)
		}
		args := []string{"build"}
		args = append(args, buildFlags...)
		args = append(args, "-o", obj)
		args = append(args, inputs...)
		if err := runCommand("go", args...); err != nil {
			return err
		}
		os.RemoveAll(tmp)
		fmt.Printf("built")
	}
	fmt.Println("\n\nbuild completed")
	return nil
}
