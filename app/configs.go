// Package main handles various utility needs
package main

import (
	"encoding/json"
	"os"
)

// ConfigFile is the injected name of the config file to read
var ConfigFile = ""

// ReadConfig will read a config file and unmarshal to JSON
func ReadConfig(obj any) error {
	b, err := os.ReadFile(ConfigFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}
