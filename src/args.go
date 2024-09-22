// Package main handles various utility needs
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type (
	// Args are common app arguments
	Args struct {
		ConfigFile string
		Flags      map[string][]string
		Name       string
		EnabledKey string
		GOOS       string
	}
)

// ReadConfig reads the argument JSON configuration file
func (a Args) ReadConfig(obj any) error {
	b, err := os.ReadFile(a.ConfigFile)
	if err != nil {
		return err
	}
	settings := make(map[string]interface{})
	if err := json.Unmarshal(b, &settings); err != nil {
		return err
	}
	sub, ok := settings["Settings"]
	if !ok {
		return fmt.Errorf("unable to find settings: %v", settings)
	}
	j, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, obj)
}
