// Package main handles various utility needs
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type (
	// Args are common app arguments
	Args struct {
		Config struct {
			Dir       string
			Extension string
		}
		Name string
	}
	// Configuration is the common core configuration
	Configuration[T any] struct {
		Flags    []string
		Settings T
	}
)

// Load will load the arguments into the configuration
func (c *Configuration[T]) Load(a Args) error {
	return c.LoadFile(filepath.Join(a.Config.Dir, fmt.Sprintf("%s%s", a.Name, a.Config.Extension)))
}

// LoadFile will load the file into the configuration
func (c *Configuration[T]) LoadFile(configFile string) error {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &c)
}
