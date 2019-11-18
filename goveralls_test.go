package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"
)

func fakeServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"error":false,"message":"Fake message","URL":"http://fake.url"}`)
	}))
}

func fakeServerWithPayloadChannel(payload chan Job) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this is a standard baked response
		fmt.Fprintln(w, `{"error":false,"message":"Fake message","URL":"http://fake.url"}`)

		body, err := ioutil.ReadAll(r.Body)
		// query params are used for the body payload
		vals, err := url.ParseQuery(string(body))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var job Job
		err = json.Unmarshal([]byte(vals["json"][0]), &job)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		payload <- job

		w.WriteHeader(http.StatusOK)
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

func TestCustomJobId(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	jobBodyChannel := make(chan Job, 8096)
	fs := fakeServerWithPayloadChannel(jobBodyChannel)

	cmd := exec.Command("goveralls", "-jobid=123abc", "-package=github.com/mattn/goveralls/tester", "-endpoint")
	cmd.Args = append(cmd.Args, "-v", "-endpoint", fs.URL)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("Expected exit code 0 got 1", err, string(b))
	}

	jobBody := <-jobBodyChannel

	if jobBody.ServiceJobId != "123abc" {
		t.Fatalf("Expected job id of 123abc, but was %s", jobBody.ServiceJobId)
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

func TestShowArg(t *testing.T) {
	tmp := prepareTest(t)
	defer os.RemoveAll(tmp)
	fs := fakeServer()

	t.Run("with show", func(t *testing.T) {
		cmd := exec.Command("goveralls", "-package=github.com/mattn/goveralls/tester/...", "-show", "-endpoint")
		cmd.Args = append(cmd.Args, "-show", "-endpoint", fs.URL)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		expected := `goveralls: github.com/mattn/goveralls/tester
Fake message
http://fake.url
`
		if string(b) != expected {
			t.Error("Unexpected output for -show:", string(b))
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
	tmp, err := ioutil.TempDir("", "goveralls")
	if err != nil {
		t.Fatal("prepareTest:", err)
	}
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
