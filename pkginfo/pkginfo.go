// Package pkginfo contains information on a package
package pkginfo

import (
	"io/ioutil"

	"github.com/apex/log"
	"gopkg.in/yaml.v2"
)

// BuildInfo contains info on building a target
type BuildInfo struct {
	// Compile lists all the sources to compile
	Compile []string

	// Link lists all the libraries to link
	Link []string
}

// TargetsInfo contains info on all targets
type TargetsInfo struct {
	// Libraries lists all the libraries to build
	Libraries map[string]BuildInfo

	// Executables lists all the executabls to build
	Executables map[string]BuildInfo
}

// TestInfo contains info on a test
type TestInfo struct {
	// Command is the command to execute
	Command string
}

// PkgInfo contains information on a package
type PkgInfo struct {
	// Name is the name of the package
	Name string

	// Dependencies are the package dependencies
	Dependencies []string

	// Targets contains information on what we need to build
	Targets TargetsInfo

	// Tests contains information on the tests to run
	Tests map[string]TestInfo
}

// Read reads package info from "MKBuild.yaml"
func Read() *PkgInfo {
	filename := "MKBuild.yaml"
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.WithError(err).Fatalf("cannot read %s", filename)
	}
	pkginfo := &PkgInfo{}
	err = yaml.Unmarshal(data, pkginfo)
	if err != nil {
		log.WithError(err).Fatalf("cannot unmarshal %s", filename)
	}
	return pkginfo
}
