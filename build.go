// Package main handles various utility needs
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

const (
	srcDir   = "src"
	appFile  = ".app.go"
	buildDir = "target"
	mainText = `// Package main handles {{ .App }}
package main

import (
	"fmt"
	"os"
)

func main() {
	args := Args{}
    {{- range $key, $value := .Variables }}
	args.{{ $key }} = "{{ $value }}"
    {{- end }}
	if err := {{ .App }}(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
`
)

var (
	configExt  = ".json"
	configDir  = filepath.Join(os.Getenv("HOME"), ".config", "etc")
	destDir    = filepath.Join(".local", "bin")
	buildFlags = []string{
		"-trimpath",
		"-buildmode=pie",
		"-mod=readonly",
		"-modcacherw",
		"-buildvcs=false",
	}
)

type (
	buildResult struct {
		name  string
		err   error
		built bool
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
	cfgs, err := os.ReadDir(configDir)
	if err != nil {
		return err
	}
	var configs []string
	for _, f := range cfgs {
		configs = append(configs, strings.TrimSuffix(f.Name(), configExt))
	}
	if len(configs) == 0 {
		return errors.New("no configs found for build targets")
	}
	files, err := os.ReadDir(srcDir)
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
			if !slices.Contains(configs, cut) {
				continue
			}
			targets = append(targets, cut)
		} else {
			source = append(source, filepath.Join(srcDir, name))
		}
	}
	tmpl, err := template.New("t").Parse(mainText)
	if err != nil {
		return err
	}
	var res []chan buildResult
	for _, target := range targets {
		r := make(chan buildResult)
		go parallelBuild(target, source, tmpl, r)
		res = append(res, r)
	}
	var errored []error
	for _, r := range res {
		result := <-r
		status := ""
		if result.err != nil {
			status = "failed"
			errored = append(errored, result.err)
		} else {
			if result.built {
				status = "built"
			} else {
				status = "up-to-date"
			}
		}
		fmt.Printf("[%s] %s\n", status, result.name)
	}
	if len(errored) > 0 {
		return errors.Join(errored...)
	}
	fmt.Println("\nbuild completed")
	return nil
}

func parallelBuild(target string, source []string, tmpl *template.Template, res chan buildResult) {
	result := buildResult{name: target}
	built, err := buildTarget(target, source, tmpl)
	if err == nil {
		result.built = built
	} else {
		result.err = err
	}
	res <- result
}

func buildTarget(target string, source []string, tmpl *template.Template) (bool, error) {
	src := []string{filepath.Join(srcDir, fmt.Sprintf("%s%s", target, appFile))}
	src = append(src, source...)
	obj := filepath.Join(buildDir, target)
	stat, err := os.Stat(obj)
	building := true
	if err == nil {
		building = false
		mod := stat.ModTime()
		checks := []string{"go.mod", "build.go"}
		checks = append(checks, src...)
		for _, f := range checks {
			info, err := os.Stat(f)
			if err != nil {
				return false, err
			}
			if info.ModTime().After(mod) {
				building = true
				break
			}
		}
	}
	if !building {
		return false, nil
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
		return false, fmt.Errorf("unable to parse target proper name: %s", target)
	}
	properName = fmt.Sprintf("%sApp", properName)
	app := struct {
		App       string
		Variables map[string]string
	}{properName, map[string]string{"Config": filepath.Join(configDir, fmt.Sprintf("%s%s", target, configExt))}}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, app); err != nil {
		return false, err
	}
	hasher := sha256.New()
	if _, err := hasher.Write([]byte(properName)); err != nil {
		return false, err
	}
	hash := hasher.Sum(nil)
	tmp := filepath.Join(buildDir, "src", fmt.Sprintf("%x", hash)[0:7])
	os.RemoveAll(tmp)
	if err := mkDirP(tmp); err != nil {
		return false, err
	}
	mainFile := filepath.Join(tmp, "main.go")
	if err := os.WriteFile(mainFile, buf.Bytes(), 0o644); err != nil {
		return false, err
	}
	inputs := []string{mainFile}
	for _, s := range src {
		name := filepath.Base(s)
		to := filepath.Join(tmp, name)
		if err := runCommand("cp", s, to); err != nil {
			return false, err
		}
		inputs = append(inputs, to)
	}
	args := []string{"build"}
	args = append(args, buildFlags...)
	args = append(args, "-o", obj)
	args = append(args, inputs...)
	return true, runCommand("go", args...)
}
