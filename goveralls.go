package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Coverage interface{}

type SourceFile struct {
	Name     string     `json:"name"`
	Source   []string   `json:"source"`
	Coverage []Coverage `json:"coverage"`
	isFile   bool
}

type Result struct {
	RepoToken        string       `json:"repo_token"`
	ServiceJobId     string       `json:"service_job_id"`
	ServiceName      string       `json:"service_name"`
	ServiceEventType string       `json:"service_event_type"`
	SourceFiles      []SourceFile `json:"source_files"`
}

var re = regexp.MustCompile("^([^/]+)/([^\\s]+)\\s+([^\\s]+)\\s+.*$")

func main() {
	if len(os.Args) == 1 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: goveralls [repo_token] [package]")
	}
	var cmd *exec.Cmd
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

	var result Result
	result.RepoToken = os.Args[1]
	result.ServiceJobId = time.Now().Format("20060102030405")
	result.ServiceEventType = "manual"

	sourceFileMap := make(map[string]SourceFile)
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
			sourceFile = SourceFile{
				Name:     file,
				Source:   []string{},
				Coverage: []Coverage{},
				isFile:   false,
			}
			sourceFileMap[file] = sourceFile
			f, err := os.Open(matches[0][2])
			if err == nil {
				b, err := ioutil.ReadAll(f)
				if err == nil {
					sourceFile.Source = strings.Split(string(b), "\n")
					sourceFile.Coverage = make([]Coverage, len(sourceFile.Source))
					sourceFile.isFile = true
				}
			}
			result.SourceFiles = append(result.SourceFiles, sourceFile)
		}

		for _, line := range strings.Split(string(ret), "\n") {
			if line != "" {
				pos := strings.Index(line, " ")
				no, err := strconv.Atoi(line[:pos])
				if err == nil {
					if no > len(sourceFile.Source) {
						for no > len(sourceFile.Source) {
							sourceFile.Source = append(sourceFile.Source, "")
							sourceFile.Coverage = append(sourceFile.Coverage, nil)
						}
						sourceFile.Source[no-1] = line[pos+5:]
					}
					if line[pos+1:pos+5] == "MISS" {
						sourceFile.Coverage[no-1] = 0
					} else {
						sourceFile.Coverage[no-1] = 1
					}
				}
			}
		}
	}

	b, err := json.Marshal(result)
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
	io.Copy(os.Stdout, res.Body)
}
