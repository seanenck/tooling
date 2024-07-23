// Package main handles media transcoding
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type (
	// Config handles tooling configuration
	Config struct {
		Transcode []Transcode
	}
	// Transcode is transode method for extensions
	Transcode struct {
		Extensions []string
		Command    []string
	}
)

func run() error {
	read, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "etc", "transcoding"))
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(read, &cfg); err != nil {
		return err
	}
	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}
	var allExtensions []string
	for _, transcode := range cfg.Transcode {
		for _, ext := range transcode.Extensions {
			if slices.Contains(allExtensions, ext) {
				return fmt.Errorf("%s is already handled", ext)
			}
			allExtensions = append(allExtensions, fmt.Sprintf(".%s", ext))
		}
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !slices.Contains(allExtensions, ext) {
			continue
		}
		ext = strings.TrimPrefix(ext, ".")
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}
		hashed := fmt.Sprintf("%x", hasher.Sum(nil))
		now := time.Now().Format("02.T_150405.")
		target := fmt.Sprintf("%s%s", now, hashed[0:7])
		done := false
		for _, transcode := range cfg.Transcode {
			if slices.Contains(transcode.Extensions, ext) {
				run := ""
				var args []string
				for idx, c := range transcode.Command {
					if idx == 0 {
						run = c
						continue
					}
					use := c
					for k, v := range map[string]string{
						"{OUTPUT}": target,
						"{INPUT}":  name,
						"{EXT}":    ext,
					} {
						use = strings.ReplaceAll(use, k, v)
					}
					args = append(args, use)
				}
				c := exec.Command(run, args...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return err
				}
				done = true
				if err := os.Remove(name); err != nil {
					return err
				}
			}
		}
		if !done {
			return fmt.Errorf("unable to transcode, no command? %s", name)
		}
	}
	return nil
}
