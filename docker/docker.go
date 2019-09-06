// Package docker implements the docker subcommand
package docker

import (
	"os"
	"text/template"

	"github.com/apex/log"
	"github.com/measurement-kit/mkbuild/pkginfo"
)

// dockerSh is the script to run a build inside a docker container.
var dockerSh = `#!/bin/sh -e
# Autogenerated by 'mkbuild'; DO NOT EDIT!

USAGE="Usage: $0 asan|clang|coverage|ubsan|vanilla"

if [ $# -eq 1 ]; then
  INTERNAL=0
  BUILD_TYPE="$1"
elif [ $# -eq 2 -a "$1" = "-internal" ]; then
  INTERNAL=1
  BUILD_TYPE="$2"
else
  echo "$USAGE" 1>&2
  exit 1
fi

if [ "$CODECOV_TOKEN" = "" ]; then
  echo "WARNING: CODECOV_TOKEN is not set" 1>&2
fi
if [ "$TRAVIS_BRANCH" = "" ]; then
  echo "WARNING: TRAVIS_BRANCH is not set" 1>&2
fi

set -x

if [ $INTERNAL -eq 0 ]; then
  exec docker run --cap-add=NET_ADMIN \
                  --cap-add=SYS_PTRACE \
                  -e CODECOV_TOKEN=$CODECOV_TOKEN \
                  -e TRAVIS_BRANCH=$TRAVIS_BRANCH \
                  -v "$(pwd):/mk" \
                  --workdir /mk \
                  -t {{.CONTAINER_NAME}} \
                  ./docker.sh -internal "$1"
fi

env | grep -v TOKEN | sort

# Select the proper build flags depending on the build type
if [ "$BUILD_TYPE" = "asan" ]; then
  export CFLAGS="-fsanitize=address -O1 -fno-omit-frame-pointer"
  export CXXFLAGS="-fsanitize=address -O1 -fno-omit-frame-pointer"
  export LDFLAGS="-fsanitize=address -fno-omit-frame-pointer"
  export CMAKE_BUILD_TYPE="Debug"

elif [ "$BUILD_TYPE" = "clang" ]; then
  export CMAKE_BUILD_TYPE="Release"
  export CXXFLAGS="-stdlib=libc++"
  export CC=clang
  export CXX=clang++

elif [ "$BUILD_TYPE" = "coverage" ]; then
  export CFLAGS="-O0 -g -fprofile-arcs -ftest-coverage"
  export CMAKE_BUILD_TYPE="Debug"
  export CXXFLAGS="-O0 -g -fprofile-arcs -ftest-coverage"
  export LDFLAGS="-lgcov"

elif [ "$BUILD_TYPE" = "ubsan" ]; then
  export CFLAGS="-fsanitize=undefined -fno-sanitize-recover"
  export CXXFLAGS="-fsanitize=undefined -fno-sanitize-recover"
  export LDFLAGS="-fsanitize=undefined"
  export CMAKE_BUILD_TYPE="Debug"

elif [ "$BUILD_TYPE" = "vanilla" ]; then
  export CMAKE_BUILD_TYPE="Release"

else
  echo "$0: BUILD_TYPE not in: asan, clang, coverage, ubsan, vanilla" 1>&2
  exit 1
fi

# Configure and make equivalent
mkdir -p build/$BUILD_TYPE
cd build/$BUILD_TYPE
cmake -GNinja -DCMAKE_BUILD_TYPE=$CMAKE_BUILD_TYPE ../../
cmake --build . -- -v

# Make sure we don't consume too much resources by bumping latency. Not all
# repositories need this feature. For them the code is commented out.
{{.TC_DISABLED}}tc qdisc add dev eth0 root netem delay 200ms 10ms

# Make check equivalent
ctest --output-on-failure -a -j8

# Stop adding latency. Commented out if we don't need it.
{{.TC_DISABLED}}tc qdisc del dev eth0 root

# Measure and possibly report the test coverage
if [ "$BUILD_TYPE" = "coverage" ]; then
  lcov --directory . --capture -o lcov.info
  if [ "$CODECOV_TOKEN" != "" ]; then
    curl -fsSL -o codecov.sh https://codecov.io/bash
    bash codecov.sh -X gcov -Z -f lcov.info
  fi
fi
`

// tcDisabledString returns an empty string is if the tc utility is configured
// to increase the latency, or a comment otherwise
func tcDisabledString(pkginfo *pkginfo.PkgInfo) (s string) {
	if pkginfo.DockerTcDisabled == true {
		s = "#"
	}
	return
}

// writeSingleDockerScript writes a single docker script.
func writeSingleDockerScript(
	pkginfo *pkginfo.PkgInfo, dirname, name, content string,
) {
	tmpl := template.Must(template.New(name).Parse(content))
	filename := dirname + "/" + name
	filep, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		log.WithError(err).Fatalf("cannot open file: %s", filename)
	}
	defer filep.Close()
	err = tmpl.Execute(filep, map[string]string{
		"CONTAINER_NAME": pkginfo.Docker,
		"TC_DISABLED":    tcDisabledString(pkginfo),
	})
	if err != nil {
		log.WithError(err).Fatalf("cannot write file: %s", filename)
	}
	log.Infof("Written %s", filename)
}

// Generate generates all the docker scripts.
func Generate(pkginfo *pkginfo.PkgInfo) {
	if pkginfo.Docker == "" {
		log.Fatal("no docker container specified")
	}
	writeSingleDockerScript(pkginfo, ".", "docker.sh", dockerSh)
}
