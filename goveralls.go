// Copyright (c) 2013 Yasuhiro Matsumoto, Jason McVetta.
// This is Free Software,  released under the MIT license.
// See http://mattn.mit-license.org/2013 for details.

// goveralls is a Go client for Coveralls.io.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

	"code.google.com/p/go-uuid/uuid"
)

/*
	https://coveralls.io/docs/api_reference
*/

var (
	pkg       = flag.String("package", "", "Go package")
	verbose   = flag.Bool("v", false, "Pass '-v' argument to 'gocov test'")
	gocovjson = flag.String("gocovdata", "", "If supplied, use existing gocov.json")
)

// usage supplants package flag's Usage variable
var usage = func() {
	cmd := os.Args[0]
	// fmt.Fprintf(os.Stderr, "Usage of %s:\n", cmd)
	s := "Usage: %s [options] TOKEN\n"
	fmt.Fprintf(os.Stderr, s, cmd)
	flag.PrintDefaults()
}

var reportRE = regexp.MustCompile(`^(\S+)/(\S+.go)\s+(\S+)\s+`)
var annotateRE = regexp.MustCompile(`^\s*(\d+) (MISS)?`)
var remotesRE = regexp.MustCompile(`^(\S+)\s+(\S+)`)

// A Head object encapsulates information about the HEAD revision of a git repo.
type Head struct {
	Id             string `json:"id"`
	AuthorName     string `json:"author_name,omitempty"`
	AuthorEmail    string `json:"author_email,omitempty"`
	CommitterName  string `json:"committer_name,omitempty"`
	CommitterEmail string `json:"committer_email,omitempty"`
	Message        string `json:"message"`
}

// A Remote object encapsulates information about a remote of a git repo.
type Remote struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// A Git object encapsulates information about a git repo.
type Git struct {
	Head    Head      `json:"head"`
	Branch  string    `json:"branch"`
	Remotes []*Remote `json:"remotes,omitempty"`
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
	RepoToken    string        `json:"repo_token"`
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

type GocovResult struct {
	Packages []struct {
		Name      string
		Functions []struct {
			Name       string
			File       string
			Start, End int
			Statements []struct {
				Start, End, Reached int
			}
		}
	}
}

// collectGitInfo runs several git commands to compose a Git object.
func collectGitInfo() *Git {
	gitCmds := map[string][]string{
		"id":      {"git", "rev-parse", "HEAD"},
		"branch":  {"git", "rev-parse", "--abbrev-ref", "HEAD"},
		"aname":   {"git", "log", "-1", "--pretty=%aN"},
		"aemail":  {"git", "log", "-1", "--pretty=%aE"},
		"cname":   {"git", "log", "-1", "--pretty=%cN"},
		"cemail":  {"git", "log", "-1", "--pretty=%cE"},
		"message": {"git", "log", "-1", "--pretty=%s"},
		"remotes": {"git", "remote", "-v"},
	}
	results := map[string]string{}
	remotes := map[string]Remote{}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		log.Fatal(err)
	}
	for key, args := range gitCmds {
		cmd := exec.Cmd{}
		cmd.Path = gitPath
		cmd.Args = args
		cmd.Stderr = os.Stderr
		ret, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		s := string(ret)
		s = strings.TrimRight(s, "\n")
		results[key] = s
	}
	for _, line := range strings.Split(results["remotes"], "\n") {
		matches := remotesRE.FindAllStringSubmatch(line, -1)
		if len(matches) != 1 {
			continue
		}
		if len(matches[0]) != 3 {
			continue
		}
		name := matches[0][1]
		url := matches[0][2]
		r := Remote{
			Name: name,
			Url:  url,
		}
		remotes[name] = r
	}
	h := Head{}
	h.Id = results["id"]
	h.AuthorName = results["aname"]
	h.AuthorEmail = results["aemail"]
	h.CommitterName = results["cname"]
	h.CommitterEmail = results["cemail"]
	h.Message = results["message"]
	g := Git{}
	g.Head = h
	g.Branch = results["branch"]
	for _, r := range remotes {
		g.Remotes = append(g.Remotes, &r)
	}
	return &g
}

func runGocov() (io.ReadCloser, error) {
	cmd := exec.Command("gocov")
	args := []string{"gocov", "test"}
	if *verbose {
		args = append(args, "-v")
	}
	if *pkg != "" {
		args = append(args, *pkg)
	}
	cmd.Args = args
	cmd.Stderr = os.Stderr
	ret, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(bytes.NewReader(ret)), nil
}

func loadCoverage() (io.ReadCloser, error) {
	if *gocovjson == "" {
		return runGocov()
	} else {
		return os.Open(*gocovjson)
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	//
	// Parse Flags
	//
	flag.Usage = usage
	service := flag.String("service", "goveralls",
		"The CI service or other environment in which the test suite was run. ")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
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
	var j Job
	j.RunAt = time.Now()
	j.RepoToken = flag.Arg(0)
	j.ServiceJobId = uuid.New()
	j.Git = collectGitInfo()
	if *service != "" {
		j.ServiceName = *service
	}

	cov, err := loadCoverage()
	if err != nil {
		log.Fatalf("Error getting coverage data: %v", err)
	}

	var result GocovResult
	d := json.NewDecoder(cov)
	err = d.Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	cov.Close()

	sourceFileMap := map[string]*SourceFile{}
	// Find all the files and load their content
	fileContent := map[string][]byte{}
	for _, pkg := range result.Packages {
		for _, fun := range pkg.Functions {
			b, ok := fileContent[fun.File]
			if !ok {
				b, err = ioutil.ReadFile(fun.File)
				if err != nil {
					log.Fatalf("Error reading %v: %v", fun.File, err)
				}
				fileContent[fun.File] = b
				// Count the lines
				sf := &SourceFile{
					Name:     fun.File,
					Source:   string(b),
					Coverage: make([]interface{}, bytes.Count(b, []byte{'\n'})),
				}
				sourceFileMap[fun.File] = sf
				j.SourceFiles = append(j.SourceFiles, sf)
			}
			sf := sourceFileMap[fun.File]

			// First, mark all parts of a mentioned function as covered.
			linenum := 0
			for i := range b {
				if i >= fun.End {
					break
				}
				if b[i] == '\n' {
					linenum++
				}
				if i >= fun.Start {
					// Leaving off a newline at the end of
					// the file can cause us to compute line
					// numbers where there are not lines.
					if linenum < len(sf.Coverage) {
						sf.Coverage[linenum] = 1
					}
				}
			}

			// Then paint each statement as directed.  This will mark misses.
			for _, st := range fun.Statements {
				linenum := 0
				for i := range b {
					if i >= st.End {
						break
					}
					if b[i] == '\n' {
						linenum++
					}
					if i >= st.Start {
						sf.Coverage[linenum] = st.Reached
						break // only count the statement start
					}
				}
			}
		}
	}

	b, err := json.Marshal(j)
	if err != nil {
		log.Fatal(err)
	}

	if j.RepoToken == "" {
		os.Stdout.Write(b)
		os.Exit(0)
	}

	params := make(url.Values)
	params.Set("json", string(b))
	res, err := http.PostForm("https://coveralls.io/api/v1/jobs", params)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	var response Response
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		log.Fatal(err)
	}
	if response.Error {
		log.Fatal(response.Message)
	}
	fmt.Println(response.Message)
	fmt.Println(response.URL)
}
