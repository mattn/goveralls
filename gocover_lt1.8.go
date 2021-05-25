// +build !go1.8

package main

import (
	"io/ioutil"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

func findRootPackage(rootDirectory string) string {
	return ""
}
