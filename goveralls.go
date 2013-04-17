package main

import (
	"encoding/json"
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

type Coverage interface{}

type SourceFile struct {
	Name     string     `json:"name"`
	Source   string     `json:"source"`
	Coverage []Coverage `json:"coverage"`
}

type Request struct {
	RepoToken        string        `json:"repo_token"`
	ServiceJobId     string        `json:"service_job_id"`
	ServiceName      string        `json:"service_name"`
	ServiceEventType string        `json:"service_event_type"`
	SourceFiles      []*SourceFile `json:"source_files"`
}

type Response struct {
	Message string `json:"message"`
	URL     string `json:"url"`
	Error   bool   `json:"error"`
}

var re = regexp.MustCompile("^([^/]+)/([^\\s]+)\\s+([^\\s]+)\\s+.*$")

func main() {
	if len(os.Args) == 1 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: goveralls [repo_token] [package]")
	}
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

	if len(os.Args) == 2 {
		cmd = exec.Command("gocov", "test")
	} else {
		cmd = exec.Command("gocov", "test", os.Args[2])
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

	var request Request
	request.RepoToken = os.Args[1]
	request.ServiceJobId = time.Now().Format("20060102030405")
	request.ServiceEventType = "manual"

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
				Coverage: []Coverage{},
			}
			sourceFileMap[file] = sourceFile
			f, err := os.Open(file)
			if err != nil {
				continue
			}
			b, err := ioutil.ReadAll(f)
			if err == nil {
				sourceFile.Source = string(b)
				sourceFile.Coverage = make([]Coverage, len(strings.Split(sourceFile.Source, "\n")))
			}
			request.SourceFiles = append(request.SourceFiles, sourceFile)
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

	b, err := json.Marshal(request)
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
