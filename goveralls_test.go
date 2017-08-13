package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/pborman/uuid"
)

func fakeServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"error":false,"message":"Fake message","URL":"http://fake.url"}`)
	}))
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

func TestInvalidArg(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	cmd := exec.Command("goveralls", "pkg")
	b, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected exit code 1 got 0")
	}
	s := strings.Split(string(b), "\n")[0]
	if !strings.HasPrefix(s, "Usage: goveralls ") {
		t.Fatalf("Expected %v, but %v", "Usage: ", s)
	}
}

func TestVerboseArg(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	fs := fakeServer()

	t.Run("with verbose", func(t *testing.T) {
		cmd := exec.Command("goveralls", "-package=github.com/mattn/goveralls/tester", "-v", "-endpoint")
		cmd.Args = append(cmd.Args, "-v", "-endpoint", fs.URL)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		if !strings.Contains(string(b), "--- PASS") {
			t.Error("Expected to have verbosed go test output in stdout", string(b))
		}
	})

	t.Run("without verbose", func(t *testing.T) {
		cmd := exec.Command("goveralls", "-package=github.com/mattn/goveralls/tester", "-endpoint")
		cmd.Args = append(cmd.Args, "-v", "-endpoint", fs.URL)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		if strings.Contains(string(b), "--- PASS") {
			t.Error("Expected to haven't verbosed go test output in stdout", string(b))
		}
	})
}

func TestRaceArg(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	fs := fakeServer()

	t.Run("it should pass the test", func(t *testing.T) {
		cmd := exec.Command("goveralls", "-package=github.com/mattn/goveralls/tester", "-race")
		cmd.Args = append(cmd.Args, "-endpoint", fs.URL)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}
	})
}

/* FIXME: currently this dones't work because the command goveralls will run
 * another session for this session.
func TestGoveralls(t *testing.T) {
	wd, _ := os.Getwd()
	tmp := prepareTest(t)
	os.Chdir(tmp)
	defer func() {
		os.Chdir(wd)
		os.RemoveAll(tmp)
	}()
	runCmd(t, "go", "get", "github.com/mattn/goveralls/testergo-runewidth")
	b := runCmd(t, "goveralls", "-package=github.com/mattn/goveralls/tester")
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	s := lines[len(lines)-1]
	if s != "Succeeded" {
		t.Fatalf("Expected test of tester are succeeded, but failured")
	}
}
*/

func prepareTest(t *testing.T) (tmpPath string) {
	tmp := os.TempDir()
	tmp = filepath.Join(tmp, uuid.New())
	runCmd(t, "go", "build", "-o", filepath.Join(tmp, "bin", "goveralls"), "github.com/mattn/goveralls")
	os.Setenv("PATH", filepath.Join(tmp, "bin")+string(filepath.ListSeparator)+os.Getenv("PATH"))
	os.MkdirAll(filepath.Join(tmp, "src"), 0755)
	return tmp
}

func runCmd(t *testing.T, cmd string, args ...string) []byte {
	b, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("Expected %v, but %v: %v", nil, err, string(b))
	}
	return b
}
