// Package main handles various utility needs
package main

import (
	"errors"
	"os"
)

// PathExists will indicate if a file/path exists
func PathExists(file string) bool {
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
