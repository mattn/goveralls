//go:build !go1.15
// +build !go1.15

package main

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestUsage(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(goverallsTestBin, "-h")
	b, err := cmd.CombinedOutput()
	runtime.Version()
	if err == nil {
		t.Fatal("Expected exit code 1 got 0")
	}
	s := strings.Split(string(b), "\n")[0]
	expectedPrefix := "Usage: goveralls"
	if runtime.GOOS == "windows" {
		expectedPrefix += ".exe"
	}
	if !strings.HasPrefix(s, expectedPrefix) {
		t.Fatalf("Expected prefix %q, but got %q", expectedPrefix, s)
	}
}
