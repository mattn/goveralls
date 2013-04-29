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
)

/*

	https://coveralls.io/docs/api_reference

*/

// A SourceFile represents a source code file and its coverage data for a
// single job.
type SourceFile struct {
	Name     string        `json:"name"`     // File path of this source file
	Source   string        `json:"source"`   // Full source code of this file
	Coverage []interface{} `json:"coverage"` // Requires both nulls and integers
}

// A Job represents the coverage data from a single run of a test suite.
type Job struct {
	RepoToken    string `json:"repo_token"`
	ServiceJobId string `json:"service_job_id"`
	ServiceName  string `json:"service_name"`
	// service_event_type seems to have been removed from the API
	// ServiceEventType string        `json:"service_event_type"`
	SourceFiles []*SourceFile `json:"source_files"`
}

type Response struct {
	Message string `json:"message"`
	URL     string `json:"url"`
	Error   bool   `json:"error"`
}

var pat = `^(\S+)/(\S+.go)\s+(\S+)\s+`
var re = regexp.MustCompile(pat)

// usage supplants package flag's Usage variable
var usage = func() {
	cmd := os.Args[0]
	// fmt.Fprintf(os.Stderr, "Usage of %s:\n", cmd)
	s := "Usage: %s [-service SERVICENAME] TOKEN"
	fmt.Fprintf(os.Stderr, s, cmd)
	flag.PrintDefaults()
}

func main() {
	//
	// Parse Flags
	//
	flag.Usage = usage
	service := flag.String("service", "", "The CI service or other environment in which the test suite was run. ")
	pkg := flag.String("package", "", "Go package")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	//
	// Run Commands
	//
	var cmd *exec.Cmd
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
	if *pkg == "" {
		cmd = exec.Command("gocov", "test")
	} else {
		cmd = exec.Command("gocov", "test", *pkg)
	}
	cmd.Stderr = os.Stderr
	ret, err := cmd.Output()
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
	//
	// Compose Job
	//
	var j Job
	j.RepoToken = flag.Arg(0)
	j.ServiceJobId = uuid.New()
	// j.ServiceEventType = "manual"
	if *service != "" {
		j.ServiceName = *service
	}
	//
	// Parse Command Output
	//
	sourceFileMap := make(map[string]*SourceFile)
	for _, line := range strings.Split(string(ret), "\n") {
		matches := re.FindAllStringSubmatch(line, -1)
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
			if line != "" {
				pos := strings.Index(line, " ")
				no, err := strconv.Atoi(line[:pos])
				if err == nil && no <= len(sourceFile.Coverage) {
					if line[pos+1:pos+5] == "MISS" {
						sourceFile.Coverage[no-1] = 0
					} else {
						sourceFile.Coverage[no-1] = 1
					}
				}
			}
		}
	}

	b, err := json.Marshal(j)
	if err != nil {
		log.Fatal(err)
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
