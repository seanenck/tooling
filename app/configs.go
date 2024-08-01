// Package main handles various utility needs
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadConfig will read a config file and unmarshal to JSON
func ReadConfig(file string, obj any) error {
	b, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "etc", fmt.Sprintf("%s.json", file)))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}
