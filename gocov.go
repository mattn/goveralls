package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

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

func runGocov() (io.ReadCloser, error) {
	cmd := exec.Command("gocov")
	args := []string{"gocov", "test"}
	if *verbose {
		args = append(args, "-v")
	}
	if *race {
		args = append(args, "-race")
	}
	args = append(args, flag.Args()...)
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

func loadGocov() (io.ReadCloser, error) {
	if *gocovjson == "" {
		return runGocov()
	} else {
		return os.Open(*gocovjson)
	}
}

func parseGocov(cov io.ReadCloser) ([]*SourceFile, error) {
	var result GocovResult
	d := json.NewDecoder(cov)
	err := d.Decode(&result)
	if err != nil {
		return nil, err
	}
	cov.Close()

	sourceFileMap := map[string]*SourceFile{}
	var rv []*SourceFile

	// Find all the files and load their content
	fileContent := map[string][]byte{}
	for _, pkg := range result.Packages {
		for _, fun := range pkg.Functions {
			b, ok := fileContent[fun.File]
			if !ok {
				b, err = ioutil.ReadFile(fun.File)
				if err != nil {
					log.Printf("Error reading %v: %v (skipping)", fun.File, err)
					continue
				}
				fileContent[fun.File] = b
				// Count the lines
				sf := &SourceFile{
					Name:     getCoverallsSourceFileName(fun.File),
					Source:   string(b),
					Coverage: make([]interface{}, bytes.Count(b, []byte{'\n'})),
				}
				sourceFileMap[fun.File] = sf
				rv = append(rv, sf)
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
	return rv, nil
}

func getCoverageGocov() []*SourceFile {
	r, err := loadGocov()
	if err != nil {
		log.Fatalf("Error loading gocov results: %v", err)
	}
	rv, err := parseGocov(r)
	if err != nil {
		log.Fatalf("Error parsing gocov: %v", err)
	}
	return rv
}
