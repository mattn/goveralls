package main

// Much of the core of this is copied from go's cover tool itself.

// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The rest is written by Dustin Sallings

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"path/filepath"

	"golang.org/x/tools/cover"
)

func findFile(file string) (string, error) {
	dir, file := filepath.Split(file)
	pkg, err := build.Import(dir, ".", build.FindOnly)
	if err != nil {
		return "", fmt.Errorf("can't find %q: %v", file, err)
	}
	return filepath.Join(pkg.Dir, file), nil
}

func parseCover(fn string) ([]*SourceFile, error) {
	profs, err := cover.ParseProfiles(fn)
	if err != nil {
		return nil, fmt.Errorf("Error parsing coverage: %v", err)
	}

	var rv []*SourceFile
	for _, prof := range profs {
		path, err := findFile(prof.FileName)
		if err != nil {
			log.Fatalf("Can't find %v", err)
		}
		fb, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Error reading %v: %v", path, err)
		}
		sf := &SourceFile{
			Name:     getCoverallsSourceFileName(path),
			Source:   string(fb),
			Coverage: make([]interface{}, 1+bytes.Count(fb, []byte{'\n'})),
		}

		for _, block := range prof.Blocks {
			for i := block.StartLine; i <= block.EndLine; i++ {
				count, _ := sf.Coverage[i-1].(int)
				sf.Coverage[i-1] = count + block.Count
			}
		}

		rv = append(rv, sf)
	}

	return rv, nil
}
