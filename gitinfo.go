package main

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

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
		if key == "branch" {
			if envBranch := os.Getenv("GIT_BRANCH"); envBranch != "" {
				results[key] = envBranch
				continue
			}
		}

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
	h := Head{
		Id:             results["id"],
		AuthorName:     results["aname"],
		AuthorEmail:    results["aemail"],
		CommitterName:  results["cname"],
		CommitterEmail: results["cemail"],
		Message:        results["message"],
	}
	g := &Git{
		Head:   h,
		Branch: results["branch"],
	}
	for _, r := range remotes {
		g.Remotes = append(g.Remotes, &r)
	}
	return g
}
