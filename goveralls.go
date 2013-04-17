package main

import (
	//"bytes"
	"encoding/json"
	"io"
	"log"
	//"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type SourceFile struct {
	Name     string        `json:"name"`
	Source   string        `json:"source"`
	Coverage []interface{} `json:"coverage"`
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
	cmd := exec.Command("gocov", "test", os.Args[2])
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
	//result.ServiceName = "travis-ci"
	result.ServiceEventType = "manual"
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
		lines := make([]string, 0)
		flags := make([]interface{}, 0)
		for _, line := range strings.Split(string(ret), "\n") {
			if line != "" {
				pos := strings.Index(line, " ")
				lines = append(lines, line[pos+6:])
				flag := line[pos+1 : pos+5]
				if flag == "MISS" {
					flags = append(flags, 0)
				} else {
					flags = append(flags, 1)
				}
			}
		}
		result.SourceFiles = append(result.SourceFiles, SourceFile{
			Name:     matches[0][2],
			Source:   strings.Join(lines, "\n"),
			Coverage: flags,
		})
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
