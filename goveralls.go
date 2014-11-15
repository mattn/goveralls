// Copyright (c) 2013 Yasuhiro Matsumoto, Jason McVetta.
// This is Free Software,  released under the MIT license.
// See http://mattn.mit-license.org/2013 for details.

// goveralls is a Go client for Coveralls.io.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
)

/*
	https://coveralls.io/docs/api_reference
*/

var (
	pkg       = flag.String("package", "", "Go package")
	verbose   = flag.Bool("v", false, "Pass '-v' argument to 'go test'")
	race      = flag.Bool("race", false, "Pass '-race' argument to 'go test'")
	gocovjson = flag.String("gocovdata", "", "If supplied, use existing gocov.json")
	coverprof = flag.String("coverprofile", "", "If supplied, use a go cover profile")
	repotoken = flag.String("repotoken", "", "Repository Token on coveralls")
	service   = flag.String("service", "travis-ci", "The CI service or other environment in which the test suite was run. ")
)

// usage supplants package flag's Usage variable
var usage = func() {
	cmd := os.Args[0]
	// fmt.Fprintf(os.Stderr, "Usage of %s:\n", cmd)
	s := "Usage: %s [options] TOKEN\n"
	fmt.Fprintf(os.Stderr, s, cmd)
	flag.PrintDefaults()
}

// A SourceFile represents a source code file and its coverage data for a
// single job.
type SourceFile struct {
	Name     string        `json:"name"`     // File path of this source file
	Source   string        `json:"source"`   // Full source code of this file
	Coverage []interface{} `json:"coverage"` // Requires both nulls and integers
}

// A Job represents the coverage data from a single run of a test suite.
type Job struct {
	RepoToken    *string       `json:"repo_token,omitempty"`
	ServiceJobId string        `json:"service_job_id"`
	ServiceName  string        `json:"service_name"`
	SourceFiles  []*SourceFile `json:"source_files"`
	Git          *Git          `json:"git,omitempty"`
	RunAt        time.Time     `json:"run_at"`
}

// A Response is returned by the Coveralls.io API.
type Response struct {
	Message string `json:"message"`
	URL     string `json:"url"`
	Error   bool   `json:"error"`
}

func getCoverage() []*SourceFile {
	if *coverprof != "" {
		return parseCover(*coverprof)
	} else {
		return getCoverageGocov()
	}
}

var vscDirs = []string{".git", ".hg", ".bzr", ".svn"}

func findRepositoryRoot(dir string) (string, bool) {
	for _, vcsdir := range vscDirs {
		if d, err := os.Stat(filepath.Join(dir, vcsdir)); err == nil && d.IsDir() {
			return dir, true
		}
	}
	nextdir := filepath.Dir(dir)
	if nextdir == dir {
		return "", false
	}
	return findRepositoryRoot(nextdir)
}

func getCoverallsSourceFileName(name string) string {
	if dir, ok := findRepositoryRoot(name); !ok {
		return name
	} else {
		filename := strings.TrimPrefix(name, dir+string(os.PathSeparator))
		return filename
	}
}

func process() error {
	log.SetFlags(log.Ltime | log.Lshortfile)
	//
	// Parse Flags
	//
	flag.Usage = usage
	flag.Parse()

	//
	// Setup PATH environment variable
	//
	paths := filepath.SplitList(os.Getenv("PATH"))
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		paths = append(paths, filepath.Join(goroot, "bin"))
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		for _, path := range filepath.SplitList(gopath) {
			paths = append(paths, filepath.Join(path, "bin"))
		}
	}
	os.Setenv("PATH", strings.Join(paths, string(filepath.ListSeparator)))

	//
	// Initialize Job
	//
	jobId := os.Getenv("TRAVIS_JOB_ID")
	if jobId == "" {
		jobId = uuid.New()
	}
	if *repotoken == "" {
		repotoken = nil // remove the entry from json
	}

	j := Job{
		RunAt:        time.Now(),
		RepoToken:    repotoken,
		ServiceJobId: jobId,
		Git:          collectGitInfo(),
		SourceFiles:  getCoverage(),
		ServiceName:  *service,
	}

	b, err := json.Marshal(j)
	if err != nil {
		return err
	}

	params := make(url.Values)
	params.Set("json", string(b))
	res, err := http.PostForm("https://coveralls.io/api/v1/jobs", params)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("Unable to read response body from coveralls: %s", err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Bad response status from coveralls: %d - %s", res.StatusCode, string(bodyBytes))
	}
	var response Response
	if err = json.Unmarshal(bodyBytes, &response); err != nil {
		return fmt.Errorf("Unable to unmarshal response JSON from coveralls: %s\n%s", err)
	}
	if response.Error {
		return errors.New(response.Message)
	}
	fmt.Println(response.Message)
	fmt.Println(response.URL)
	return nil
}

func main() {
	if err := process(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
