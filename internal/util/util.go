package util

import (
	"os"
)

func Getwd() string {
	if cwd, err := os.Getwd(); err != nil {
		return ""
	} else {
		return cwd
	}
}
