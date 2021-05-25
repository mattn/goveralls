package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"
)

var goverallsTestBin string

func TestMain(m *testing.M) {
	tmpBin, err := ioutil.TempDir("", "goveralls_")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// generate the test binary used by all tests
	goverallsTestBin = filepath.Join(tmpBin, "bin", "goveralls")
	if runtime.GOOS == "windows" {
		goverallsTestBin += ".exe"
	}
	_, err = exec.Command("go", "build", "-o", goverallsTestBin, ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// run all tests
	exitVal := m.Run()

	os.RemoveAll(tmpBin)

	os.Exit(exitVal)
}

func fakeServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"error":false,"message":"Fake message","URL":"http://fake.url"}`)
	}))
}

func fakeServerWithPayloadChannel(payload chan Job) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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
		// this is a standard baked response
		fmt.Fprintln(w, `{"error":false,"message":"Fake message","URL":"http://fake.url"}`)
	}))
}

func TestCustomJobId(t *testing.T) {
	t.Parallel()

	jobBodyChannel := make(chan Job, 16)
	fs := fakeServerWithPayloadChannel(jobBodyChannel)

	b, err := testRun("-jobid=123abc", "-package=github.com/mattn/goveralls/tester", "-endpoint", "-v", "-endpoint", fs.URL)
	if err != nil {
		t.Fatal("Expected exit code 0 got 1", err, string(b))
	}

	jobBody := <-jobBodyChannel

	if jobBody.ServiceJobID != "123abc" {
		t.Fatalf("Expected job id of 123abc, but was %s", jobBody.ServiceJobID)
	}
}

func TestInvalidArg(t *testing.T) {
	t.Parallel()

	b, err := testRun("pkg")
	if err == nil {
		t.Fatal("Expected exit code 1 got 0")
	}
	s := strings.Split(string(b), "\n")[0]
	expectedPrefix := "Usage: goveralls"
	if !strings.HasPrefix(s, expectedPrefix) {
		t.Fatalf("Expected %q, but got %q", expectedPrefix, s)
	}
}

func TestVerboseArg(t *testing.T) {
	t.Parallel()

	fs := fakeServer()

	t.Run("with verbose", func(t *testing.T) {
		t.Parallel()

		b, err := testRun("-package=github.com/mattn/goveralls/tester", "-v", "-endpoint", "-v", "-endpoint", fs.URL)
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		if !strings.Contains(string(b), "--- PASS") {
			t.Error("Expected to have verbose go test output in stdout", string(b))
		}
	})

	t.Run("without verbose", func(t *testing.T) {
		t.Parallel()

		b, err := testRun("-package=github.com/mattn/goveralls/tester", "-endpoint", "-v", "-endpoint", fs.URL)
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		if strings.Contains(string(b), "--- PASS") {
			t.Error("Expected to haven't verbose go test output in stdout", string(b))
		}
	})
}

func TestShowArg(t *testing.T) {
	t.Parallel()

	fs := fakeServer()

	t.Run("with show", func(t *testing.T) {
		t.Parallel()

		b, err := testRun("-package=github.com/mattn/goveralls/tester/...", "-show", "-endpoint", "-show", "-endpoint", fs.URL)
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
	t.Parallel()

	fs := fakeServer()

	t.Run("it should pass the test", func(t *testing.T) {
		t.Parallel()

		b, err := testRun("-package=github.com/mattn/goveralls/tester", "-race", "-endpoint", fs.URL)
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}
	})
}

func TestUploadSource(t *testing.T) {
	t.Parallel()

	t.Run("with uploadsource", func(t *testing.T) {
		t.Parallel()

		jobBodyChannel := make(chan Job, 16)
		fs := fakeServerWithPayloadChannel(jobBodyChannel)

		b, err := testRun("-uploadsource=true", "-package=github.com/mattn/goveralls/tester", "-endpoint", "-v", "-endpoint", fs.URL)
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		jobBody := <-jobBodyChannel

		for _, sf := range jobBody.SourceFiles {
			if len(sf.Source) == 0 {
				t.Fatalf("expected source for %q to not be empty", sf.Name)
			}
		}
	})

	t.Run("without uploadsource", func(t *testing.T) {
		t.Parallel()

		jobBodyChannel := make(chan Job, 16)
		fs := fakeServerWithPayloadChannel(jobBodyChannel)

		b, err := testRun("-uploadsource=false", "-package=github.com/mattn/goveralls/tester", "-endpoint", "-v", "-endpoint", fs.URL)
		if err != nil {
			t.Fatal("Expected exit code 0 got 1", err, string(b))
		}

		jobBody := <-jobBodyChannel
		for _, sf := range jobBody.SourceFiles {
			if len(sf.Source) != 0 {
				t.Fatalf("expected source for %q to be empty", sf.Name)
			}
		}
	})
}

func testRun(args ...string) ([]byte, error) {
	// always disallow the git fetch automatically used for GitHub Actions
	args = append([]string{"-allowgitfetch=false"}, args...)
	return exec.Command(goverallsTestBin, args...).CombinedOutput()
}
