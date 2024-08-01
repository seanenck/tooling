// Package main handles media transcoding
package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// TranscodeMediaApp handles transcoding of media to other formats in mass
func TranscodeMediaApp() error {
	cfg := struct {
		Transcode []struct {
			Extensions []string
			Command    []string
		}
	}{}
	if err := ReadConfig(&cfg); err != nil {
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
				fmt.Printf("  %s -> %s\n", name, target)
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
