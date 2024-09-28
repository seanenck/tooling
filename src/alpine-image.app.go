package main

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// AlpineImageApp helps build a personal alpine image
func AlpineImageApp(a Args) error {
	type repo struct {
		URL          string
		Repositories []string
	}
	type source struct {
		Remote    string
		Directory string
		Template  []string
	}
	cfg := struct {
		Repository   repo
		Architecture []string
		Name         string
		Source       source
		Output       string
		World        string
		Timestamp    string
	}{}
	if err := a.ReadConfig(&cfg); err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "alpine-iso")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	var arch string
	switch len(cfg.Architecture) {
	case 0:
		return errors.New("unable to read architecture, nothing given")
	case 1:
		arch = cfg.Architecture[0]
	default:
		b, err := exec.Command(cfg.Architecture[0], cfg.Architecture[1:]...).CombinedOutput()
		if err != nil {
			return err
		}
		arch = strings.TrimSpace(string(b))
	}
	var tag string
	flags := os.Args
	switch len(flags) {
	case 1:
		i, err := readFileToTrimmedLines("/etc/os-release")
		if err != nil {
			return err
		}
		for line := range i {
			v, ok := strings.CutPrefix(line, "VERSION_ID=")
			if ok {
				tag = v
				break
			}
		}
		if tag == "" {
			return errors.New("unable to parse current os tag")
		}

	case 2:
		tag = flags[1]
	default:
		return fmt.Errorf("unknown arguments, too many: %v", flags)
	}
	rawTag := tag
	if strings.Contains(rawTag, ".") {
		tag = fmt.Sprintf("v%s", strings.Join(strings.Split(tag, ".")[0:2], "."))
	} else {
		if rawTag != "edge" {
			return fmt.Errorf("unknown version/not edge: %s", rawTag)
		}
	}
	i, err := readFileToTrimmedLines(cfg.World)
	if err != nil {
		return err
	}
	var world []string
	for line := range i {
		if line != "" {
			world = append(world, line)
		}
	}
	obj := struct {
		Name  string
		Arch  string
		World string
		Tag   string
	}{cfg.Name, arch, strings.Join(world, " "), tag}
	t, err := template.New("t").Parse(strings.Join(cfg.Source.Template, "\n"))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, obj); err != nil {
		return err
	}
	var repositories []string
	url := cfg.Repository.URL
	for _, repo := range cfg.Repository.Repositories {
		r, err := template.New("t").Parse(fmt.Sprintf("%s/%s", url, repo))
		if err != nil {
			return err
		}
		var text bytes.Buffer
		if err := r.Execute(&text, obj); err != nil {
			return err
		}

		repositories = append(repositories, "--repository", text.String())
	}

	clone := filepath.Join(dir, "aports")
	cmd := exec.Command("git", "clone", "--depth=1", cfg.Source.Remote, clone)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	root := filepath.Join(clone, cfg.Source.Directory)
	profile := filepath.Join(root, fmt.Sprintf("mkimg.%s.sh", cfg.Name))
	if err := os.WriteFile(profile, buf.Bytes(), 0o755); err != nil {
		return err
	}

	out := filepath.Join(dir, "iso")
	if err := os.Mkdir(out, 0o755); err != nil {
		return err
	}
	args := []string{
		filepath.Join(root, "mkimage.sh"),
		"--outdir", out,
		"--arch", arch,
		"--profile", cfg.Name,
		"--tag", rawTag,
	}
	args = append(args, repositories...)
	fmt.Println(args)
	cmd = exec.Command("sh", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	files, err := os.ReadDir(out)
	if err != nil {
		return nil
	}
	found := false
	home := os.Getenv("HOME")
	to := filepath.Join(home, cfg.Output)
	now := time.Now().Format(cfg.Timestamp)
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, ".iso") {
			fmt.Printf("artifact: %s\n", name)
			found = true
			dest := filepath.Join(to, fmt.Sprintf("%s.%s", now, name))
			if err := exec.Command("mv", filepath.Join(out, name), dest).Run(); err != nil {
				return err
			}
		}
	}
	if !found {
		return errors.New("no built iso found?")
	}
	return nil
}

func readFileToTrimmedLines(file string) (iter.Seq[string], error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	text := strings.Split(string(b), "\n")
	return func(yield func(f string) bool) {
		for _, line := range text {
			t := strings.TrimSpace(line)
			if !yield(t) {
				return
			}
		}
	}, nil
}
