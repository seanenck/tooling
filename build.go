// Package main handles various utility needs
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
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
	configExt  = ".json"
	enabledKey = "enabled"
	srcDir     = "src"
	appFile    = ".app.go"
	buildDir   = "target"
	mainText   = `// Package main handles {{ .App }}
package main

import (
	"fmt"
	"os"
)

func main() {
	args := Args{}
    {{- range $key, $value := .Variables }}
	args.{{ $key }} = {{ if not $value.Raw }}"{{ end }}{{ $value.Value }}{{ if not $value.Raw }}"{{ end }}
    {{- end }}
	if err := {{ .App }}(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
`
)

var (
	configFiles = filepath.Join(os.Getenv("HOME"), ".config", "tooling")
	destDir     = filepath.Join("/opt", "fs", "root", "bin")
	buildFlags  = []string{
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
		fmt.Fprintf(os.Stderr, "\n===\nbuild failed: %v\n", err)
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
		files, err := os.ReadDir(buildDir)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			to := filepath.Join(destDir, name)
			fmt.Printf("install -> %s (destination: %s)\n", name, to)
			if err := runCommand("install", "-m755", filepath.Join(buildDir, name), to); err != nil {
				return err
			}
		}
		return nil
	}
	var configs []string
	var targetFlags []string
	targetFlags = append(targetFlags, "make(map[string][]string)")
	dir, err := os.ReadDir(configFiles)
	if err != nil {
		return err
	}
	for _, d := range dir {
		name := d.Name()
		target, ok := strings.CutSuffix(name, configExt)
		if !ok {
			continue
		}
		b, err := os.ReadFile(filepath.Join(configFiles, name))
		if err != nil {
			return err
		}

		cfg := make(map[string]interface{})
		if err := json.Unmarshal(b, &cfg); err != nil {
			return err
		}
		set, ok := cfg["Flags"]
		if !ok {
			return fmt.Errorf("invalid settings json, no flags: %s", name)
		}
		flags, ok := set.([]interface{})
		if !ok {
			return fmt.Errorf("invalid settings json, flags array is invalid: %s", name)
		}
		var setFlags []string
		for _, f := range flags {
			s, ok := f.(string)
			if !ok {
				return fmt.Errorf("%v is not string: %s", f, name)
			}
			if s == enabledKey {
				configs = append(configs, target)
			}
			setFlags = append(setFlags, fmt.Sprintf("\"%s\"", s))
		}
		targetFlags = append(targetFlags, fmt.Sprintf("\targs.Flags[\"%s\"] = []string{%s}", target, strings.Join(setFlags, ", ")))
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
	flags := strings.Join(targetFlags, "\n")
	var res []chan buildResult
	for _, target := range targets {
		r := make(chan buildResult)
		go parallelBuild(target, flags, source, tmpl, r)
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

func parallelBuild(target, flags string, source []string, tmpl *template.Template, res chan buildResult) {
	result := buildResult{name: target}
	built, err := buildTarget(target, flags, source, tmpl)
	if err == nil {
		result.built = built
	} else {
		result.err = err
	}
	res <- result
}

func buildTarget(target, flags string, source []string, tmpl *template.Template) (bool, error) {
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
	type variable struct {
		Value string
		Raw   bool
	}
	app := struct {
		App       string
		Variables map[string]variable
	}{properName, map[string]variable{
		"Name":       {Value: target},
		"ConfigFile": {Value: filepath.Join(configFiles, fmt.Sprintf("%s%s", target, configExt))},
		"Flags":      {Value: flags, Raw: true},
		"EnabledKey": {Value: enabledKey},
	}}
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
