package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.google.com/p/go-uuid/uuid"
)

func myImportPath() string {
	cmd := exec.Command("go", "list")
	b, err := cmd.CombinedOutput()
	if err == nil {
		panic(err)
	}
	return strings.TrimSpace(string(b))
}

func TestUsage(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
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
	tmp := prepareTest(t)
	p := myImportPath()
	defer os.RemoveAll(tmp)
	runCmd(t, "go", "get", p+"/tester")
	runCmd(t, "go", "get", "github.com/axw/gocov/gocov")
	b := runCmd(t, "./goveralls", "-package="+p+"/tester", "")
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	s := lines[len(lines)-1]
	if s != "Succeeded" {
		t.Fatalf("Expected test of tester are succeeded, but failured")
	}
}

func TestGoverallsExisting(t *testing.T) {
	p := myImportPath()
	t.Logf("My import path is %q", p)

	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	runCmd(t, "go", "get", p+"/tester")
	runCmd(t, "go", "get", "github.com/axw/gocov/gocov")
	b := runCmd(t, "goveralls", "-gocovdata=tester/cov.json",
		"-package="+p+"/tester", "")
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	s := lines[len(lines)-1]
	if s != "Succeeded" {
		t.Fatalf("Expected test of tester are succeeded, but failured")
	}
}

func prepareTest(t *testing.T) (tmpPath string) {
	tmp := os.TempDir()
	tmp = filepath.Join(tmp, uuid.New())
	os.Setenv("GOPATH", tmp)
	path := os.Getenv("PATH")
	path = tmp + "/bin:" + path
	os.Setenv("PATH", path)
	runCmd(t, "go", "get", myImportPath())
	return tmp
}

func runCmd(t *testing.T, cmd string, args ...string) []byte {
	b, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("Expected %v, but %v: %v", nil, err, string(b))
	}
	return b
}
