package main

import (
	"code.google.com/p/go-uuid/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsage(t *testing.T) {
	cmd := exec.Command("goveralls", "-h")
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected exit code 1 bot 0")
	}
	s := strings.Split(string(b), "\n")[0]
	if !strings.HasPrefix(s, "Usage: goveralls ") {
		t.Fatalf("Expected %v, but %v", "Usage: ", s)
	}
}

func TestGoveralls(t *testing.T) {
	tmp := os.TempDir()
	tmp = filepath.Join(tmp, uuid.New())
	gopath := os.Getenv("GOPATH")
	os.Setenv("GOPATH", tmp)
	defer func() {
		os.RemoveAll(tmp)
		os.Setenv("GOPATH", gopath)
	}()

	b, err := exec.Command("go", "get", "github.com/mattn/goveralls/tester").CombinedOutput()
	if err != nil {
		t.Fatalf("Expected %v, but %v: %v", nil, err, string(b))
	}
	b, err = exec.Command("go", "get", "github.com/axw/gocov").CombinedOutput()
	if err != nil {
		t.Fatalf("Expected %v, but %v: %v", nil, err, string(b))
	}
	cmd := exec.Command("goveralls", "-package=github.com/mattn/goveralls/tester", "")
	b, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Expected %v, but %v", nil, err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	s := lines[len(lines)-1]
	if s != "Succeeded" {
		t.Fatalf("Expected test of tester are succeeded, but failured")
	}
}
