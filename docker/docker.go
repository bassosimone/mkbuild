// Package docker implements the docker subcommand
package docker

import (
	"os"
	"text/template"

	"github.com/apex/log"
	"github.com/bassosimone/mkbuild/pkginfo"
)

// runSh is the script that will run the test.
var runSh = `#!/bin/sh -e
USAGE="Usage: $0 asan|clang|coverage|tsan|ubsan|vanilla"

if [ $# -ne 1 ]; then
  echo "$USAGE" 1>&2
  exit 1
fi
BUILD_TYPE="$1"
shift

if [ "$CODECOV_TOKEN" = "" ]; then
  echo "WARNING: CODECOV_TOKEN is not set" 1>&2
fi
if [ "$TRAVIS_BRANCH" = "" ]; then
  echo "WARNING: TRAVIS_BRANCH is not set" 1>&2
fi
set -x

cd /mk
env | grep -v TOKEN | sort

# Make sure we don't consume too much resources by bumping latency
tc qdisc add dev eth0 root netem delay 200ms 10ms

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
  echo "$0: BUILD_TYPE not in: asan, clang, coverage, tsan, ubsan, vanilla" 1>&2
  exit 1
fi

# Configure, make, and make check equivalent
cmake -GNinja -DCMAKE_BUILD_TYPE=$CMAKE_BUILD_TYPE .
cmake --build . -- -v
ctest --output-on-failure -a -j8

# Measure and possibly report the test coverage
if [ "$BUILD_TYPE" = "coverage" ]; then
  lcov --directory . --capture -o lcov.info
  if [ "$CODECOV_TOKEN" != "" ]; then
    curl -fsSL -o codecov.sh https://codecov.io/bash
    bash codecov.sh -X gcov -Z -f lcov.info
  fi
fi
`

// trampolineSh is the script that will run docker
var trampolineSh = `#!/bin/sh -e
docker run --cap-add=NET_ADMIN \
          -e CODECOV_TOKEN=$CODECOV_TOKEN \
          -e TRAVIS_BRANCH=$TRAVIS_BRANCH \
          -v "$(pwd):/mk" \
          -t {{.CONTAINER_NAME}} \
          /mk/.ci/docker/run.sh
`

// writeDockerScripts writes the docker scripts.
func writeDockerScripts(pkginfo *pkginfo.PkgInfo) {
	dirname := ".ci/docker"
	err := os.MkdirAll(dirname, 0755)
	if err != nil {
		log.WithError(err).Fatalf("cannot create dir: %s", dirname)
	}
	{
		tmpl := template.Must(template.New("run.sh").Parse(runSh))
		filename := dirname + "/run.sh"
		filep, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			log.WithError(err).Fatalf("cannot open file: %s", filename)
		}
		defer filep.Close()
		err = tmpl.Execute(filep, map[string]string{})
		if err != nil {
			log.WithError(err).Fatalf("cannot write file: %s", filename)
		}
	}
	{
		tmpl := template.Must(template.New("trampoline.sh").Parse(trampolineSh))
		filename := dirname + "/trampoline.sh"
		filep, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			log.WithError(err).Fatalf("cannot open file: %s", filename)
		}
		defer filep.Close()
		err = tmpl.Execute(filep, map[string]string{
			"CONTAINER_NAME": pkginfo.Docker,
		})
		if err != nil {
			log.WithError(err).Fatalf("cannot write file: %s", filename)
		}
	}
}

// Run implements the docker subcommand.
func Run(pkginfo *pkginfo.PkgInfo, buildType string) {
	if pkginfo.Docker == "" {
		log.Fatal("no docker container specified")
	}
	writeDockerScripts(pkginfo)
}
