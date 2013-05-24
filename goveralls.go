// Copyright (c) 2013 Yasuhiro Matsumoto, Jason McVetta.
// This is Free Software,  released under the MIT license.
// See http://mattn.mit-license.org/2013 for details.

// goveralls is a Go client for Coveralls.io.
package main

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
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
	"strconv"
	"strings"
	"time"
)

/*
	https://coveralls.io/docs/api_reference
*/

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
		Name string
		Functions []struct {
			Name string
			File string
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

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	//
	// Parse Flags
	//
	flag.Usage = usage
	service := flag.String("service", "goveralls", "The CI service or other environment in which the test suite was run. ")
	pkg := flag.String("package", "", "Go package")
	verbose := flag.Bool("v", false, "Pass '-v' argument to 'gocov test'")
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
	//
	// Run gocov
	//
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
		log.Fatal(err)
	}

	var result GocovResult
	err = json.Unmarshal(ret, &result)
	if err != nil {
		log.Fatal(err)
	}

	covret := string(ret)
	cmd = exec.Command("gocov", "report")
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(string(ret))
	ret, err = cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	sourceFileMap := make(map[string]*SourceFile)
	for _, line := range strings.Split(string(ret), "\n") {
		matches := reportRE.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		cmd = exec.Command("gocov", "annotate", "-", matches[0][1]+"."+matches[0][3])
		cmd.Stderr = os.Stderr
		cmd.Stdin = strings.NewReader(covret)
		ret, err = cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		file := matches[0][2]
		for _, pkg := range result.Packages {
			for _, fnc := range pkg.Functions {
				if fnc.Name == matches[0][3] {
					file = fnc.File
				}
			}
		}
		sourceFile, ok := sourceFileMap[file]
		if !ok {
			sourceFile = &SourceFile{
				Name:     file,
				Source:   "",
				Coverage: []interface{}{},
			}
			sourceFileMap[file] = sourceFile
			f, err := os.Open(file)
			if err != nil {
				log.Fatal(err)
			}
			b, err := ioutil.ReadAll(f)
			if err == nil {
				sourceFile.Source = string(b)
				sourceFile.Coverage = make([]interface{}, len(strings.Split(sourceFile.Source, "\n")))
			}
			j.SourceFiles = append(j.SourceFiles, sourceFile)
		}
		for _, line := range strings.Split(string(ret), "\n") {
			matches := annotateRE.FindAllStringSubmatch(line, -1)
			if len(matches) == 0 {
				continue
			}
			numStr := matches[0][1]
			miss := matches[0][2]
			num, err := strconv.Atoi(numStr)
			if err != nil {
				log.Fatal(err)
			}
			if num > len(sourceFile.Coverage) {
				log.Panic("How did we get here??")
			}
			if miss == "MISS" {
				sourceFile.Coverage[num-1] = 0
			} else {
				sourceFile.Coverage[num-1] = 1
			}
		}
	}

	b, err := json.Marshal(j)
	if err != nil {
		log.Fatal(err)
	}

	if j.RepoToken == "" {
		fmt.Println("Succeeded")
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
