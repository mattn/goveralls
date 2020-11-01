// +build !go1.15

package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestUsage(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	cmd := exec.Command("goveralls", "-h")
	b, err := cmd.CombinedOutput()
	runtime.Version()
	if err == nil {
		t.Fatal("Expected exit code 1 got 0")
	}
	s := strings.Split(string(b), "\n")[0]
	if !strings.HasPrefix(s, "Usage: goveralls ") {
		t.Fatalf("Expected %v, but %v", "Usage: ", s)
	}
}
