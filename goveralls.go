// Copyright (c) 2013 Yasuhiro Matsumoto, Jason McVetta.
// This is Free Software,  released under the MIT license.
// See http://mattn.mit-license.org/2013 for details.

// goveralls is a Go client for Coveralls.io.
package main

import (
	_ "crypto/sha512"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

/*
	https://coveralls.io/docs/api_reference
*/

var (
	pkg       = flag.String("package", "", "Go package")
	verbose   = flag.Bool("v", false, "Pass '-v' argument to 'go test'")
	coverprof = flag.String("coverprofile", "", "If supplied, use a go cover profile")
	covermode = flag.String("covermode", "count", "sent as covermode argument to go test")
	repotoken = flag.String("repotoken", os.Getenv("COVERALLS_TOKEN"), "Repository Token on coveralls")
	endpoint  = flag.String("endpoint", "https://coveralls.io", "Hostname to submit Coveralls data to")
	service   = flag.String("service", "travis-ci", "The CI service or other environment in which the test suite was run. ")
	shallow   = flag.Bool("shallow", false, "Shallow coveralls internal server errors")
	ignore    = flag.String("ignore", "", "Comma separated files to ignore")
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
	RepoToken          *string       `json:"repo_token,omitempty"`
	ServiceJobId       string        `json:"service_job_id"`
	ServicePullRequest string        `json:"service_pull_request,omitempty"`
	ServiceName        string        `json:"service_name"`
	SourceFiles        []*SourceFile `json:"source_files"`
	Git                *Git          `json:"git,omitempty"`
	RunAt              time.Time     `json:"run_at"`
}

// A Response is returned by the Coveralls.io API.
type Response struct {
	Message string `json:"message"`
	URL     string `json:"url"`
	Error   bool   `json:"error"`
}

func getCoverage() ([]*SourceFile, error) {
	if *coverprof != "" {
		return parseCover(*coverprof)
	}

	f, err := ioutil.TempFile("", "goveralls")
	if err != nil {
		return nil, err
	}
	f.Close()
	defer os.Remove(f.Name())

	cmd := exec.Command("go")
	args := []string{"go", "test", "-covermode", *covermode, "-coverprofile", f.Name()}
	if *verbose {
		args = append(args, "-v")
	}
	args = append(args, flag.Args()...)
	if *pkg != "" {
		args = append(args, *pkg)
	}
	cmd.Args = args
	b, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %v", err, string(b))
	}
	return parseCover(f.Name())
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
	var jobId string
	if travisJobId := os.Getenv("TRAVIS_JOB_ID"); travisJobId != "" {
		jobId = travisJobId
	} else if circleCiJobId := os.Getenv("CIRCLE_BUILD_NUM"); circleCiJobId != "" {
		jobId = circleCiJobId
	} else {
		jobId = uuid.New()
	}
	if *repotoken == "" {
		repotoken = nil // remove the entry from json
	}
	var pullRequest string
	if prNumber := os.Getenv("CIRCLE_PR_NUMBER"); prNumber != "" {
		// for Circle CI (pull request from forked repo)
		pullRequest = prNumber
	} else if prNumber := os.Getenv("TRAVIS_PULL_REQUEST"); prNumber != "" && prNumber != "false" {
		pullRequest = prNumber
	} else if prURL := os.Getenv("CI_PULL_REQUEST"); prURL != "" {
		// for Circle CI
		pullRequest = regexp.MustCompile(`[0-9]+$`).FindString(prURL)
	}

	sourceFiles, err := getCoverage()
	if err != nil {
		return err
	}

	j := Job{
		RunAt:              time.Now(),
		RepoToken:          repotoken,
		ServiceJobId:       jobId,
		ServicePullRequest: pullRequest,
		Git:                collectGitInfo(),
		SourceFiles:        sourceFiles,
		ServiceName:        *service,
	}

	// Ignore files
	if len(*ignore) > 0 {
		patterns := strings.Split(*ignore, ",")
		for i, pattern := range patterns {
			patterns[i] = strings.TrimSpace(pattern)
		}
		var files []*SourceFile
	Files:
		for _, file := range j.SourceFiles {
			for _, pattern := range patterns {
				match, err := filepath.Match(pattern, file.Name)
				if err != nil {
					return err
				}
				if match {
					fmt.Printf("ignoring %s\n", file.Name)
					continue Files
				}
			}
			files = append(files, file)
		}
		j.SourceFiles = files
	}

	b, err := json.Marshal(j)
	if err != nil {
		return err
	}

	params := make(url.Values)
	params.Set("json", string(b))
	res, err := http.PostForm(*endpoint+"/api/v1/jobs", params)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("Unable to read response body from coveralls: %s", err)
	}

	if res.StatusCode >= http.StatusInternalServerError && *shallow {
		fmt.Println("coveralls server failed internally")
		return nil
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
