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
	}
)

// ReadConfig reads the argument JSON configuration file
func (a Args) ReadConfig(obj any) error {
	res, err := a.settings()
	if err != nil {
		return err
	}
	sub, ok := res[a.Name]
	if !ok {
		return fmt.Errorf("missing settings for %s", a.Name)
	}
	s, ok := sub.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid json for %s", a.Name)
	}
	m, ok := s["Settings"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid json for %s, no settings", a.Name)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, obj)
}

func (a Args) settings() (map[string]interface{}, error) {
	b, err := os.ReadFile(a.ConfigFile)
	if err != nil {
		return nil, err
	}
	res := make(map[string]interface{})
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	return res, nil
}
