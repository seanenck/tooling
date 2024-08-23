// Package main handles various utility needs
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var (
	// ConfigPath is set by applications to the location of config file
	ConfigPath = ""
	// ConfigExtension is the file extension for configs
	ConfigExtension = ""
)

// ReadConfig will read a config file and unmarshal to JSON
func ReadConfig(obj any) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(ConfigPath, fmt.Sprintf("%s%s", filepath.Base(exe), ConfigExtension)))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}
