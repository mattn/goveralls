package tester

import (
	"os"
)

func GoverallsTester() string {
	s := os.Getenv("GOVERALLS_TESTER")
	if s == "" {
		s = "hello world"
	}
	return s
}
