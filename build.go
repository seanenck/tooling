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
	"runtime"
	"slices"
	"strings"
	"text/template"
)

const (
	destDir   = "DESTDIR"
	configExt = ".json"
	srcDir    = "src"
	appFile   = ".app.go"
	mainText  = `// Package main handles {{ .App }}
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	args := Args{}
    {{- range $key, $value := .Variables }}
	args.{{ $key }} = {{ if not $value.Raw }}"{{ end }}{{ $value.Value }}{{ if not $value.Raw }}"{{ end }}
    {{- end }}
	if err := runApp(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func runApp(args Args) error {
	if args.GOOS != runtime.GOOS {
		return fmt.Errorf("unable to run on this OS")
	}
	return {{ .App }}(args)
}
`
)

var (
	configOffset = filepath.Join(".config", "tooling")
	buildFlags   = []string{
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
	buildRequest struct {
		target   string
		flags    string
		buildDir string
		goos     string
		sources  []string
		tmpl     *template.Template
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
	offset := 0
	switch len(args) {
	case 1:
	case 2:
		offset++
		arg := args[1]
		switch arg {
		default:
			return fmt.Errorf("unknown argument: %s", arg)
		}
	}
	goos := os.Getenv("OS")
	if goos == "" {
		goos = runtime.GOOS
	}
	if err := os.Setenv("GOOS", goos); err != nil {
		return err
	}
	buildDir := os.Getenv("BUILDDIR")
	if err := mkDirP(buildDir); err != nil {
		return err
	}
	var configs []string
	var targetFlags []string
	installs := []string{fmt.Sprintf("%s := %s", destDir, filepath.Join("$(HOME)", ".local", "bin")), "all:"}
	targetFlags = append(targetFlags, "make(map[string][]string)")
	configFiles := filepath.Join(os.Getenv("HOME"), configOffset)
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
		isEnabled := false
		for _, f := range flags {
			s, ok := f.(string)
			if !ok {
				return fmt.Errorf("%v is not string: %s", f, name)
			}
			switch s {
			case goos, "all":
				isEnabled = true
				configs = append(configs, target)
				installs = append(installs, fmt.Sprintf("\tinstall -m755 %s %s", target, filepath.Join(fmt.Sprintf("$(%s)", destDir), target)))
			}
			setFlags = append(setFlags, fmt.Sprintf("\"%s\"", s))
		}
		if !isEnabled {
			continue
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
		ask := buildRequest{target, flags, buildDir, goos, source, tmpl}
		go parallelBuild(ask, r)
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
	return os.WriteFile(filepath.Join(buildDir, "Makefile"), []byte(strings.Join(installs, "\n")), 0o644)
}

func parallelBuild(ask buildRequest, res chan buildResult) {
	result := buildResult{name: ask.target}
	built, err := buildTarget(ask)
	if err == nil {
		result.built = built
	} else {
		result.err = err
	}
	res <- result
}

func buildTarget(ask buildRequest) (bool, error) {
	src := []string{filepath.Join(srcDir, fmt.Sprintf("%s%s", ask.target, appFile))}
	src = append(src, ask.sources...)
	obj := filepath.Join(ask.buildDir, ask.target)
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
	for _, r := range ask.target {
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
		return false, fmt.Errorf("unable to parse target proper name: %s", ask.target)
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
		"Name":       {Value: ask.target},
		"ConfigFile": {Value: fmt.Sprintf("filepath.Join(os.Getenv(\"HOME\"), \"%s\")", filepath.Join(configOffset, fmt.Sprintf("%s%s", ask.target, configExt))), Raw: true},
		"Flags":      {Value: ask.flags, Raw: true},
		"GOOS":       {Value: ask.goos},
	}}
	var buf bytes.Buffer
	if err := ask.tmpl.Execute(&buf, app); err != nil {
		return false, err
	}
	hasher := sha256.New()
	if _, err := hasher.Write([]byte(properName)); err != nil {
		return false, err
	}
	hash := hasher.Sum(nil)
	tmp := filepath.Join(ask.buildDir, "src", fmt.Sprintf("%x", hash)[0:7])
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
