// Package main handles various utility needs
package main

import (
	"encoding/json"
	"os"
)

type (
	// Args are common app arguments
	Args struct {
		Config string
	}
)

// ReadConfig reads the argument JSON configuration file
func (a Args) ReadConfig(obj any) error {
	b, err := os.ReadFile(a.Config)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}
