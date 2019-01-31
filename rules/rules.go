// Package rules contains the build rules.
package rules

import (
	"fmt"
	"path/filepath"

	"github.com/bassosimone/mkbuild/cmake"
)

// WriteSectionComment writes a comment for |name| in |cmake|.
func WriteSectionComment(cmake *cmake.CMake, name string) {
	cmake.WriteLine("")
	cmake.WriteLine(fmt.Sprintf("#"))
	cmake.WriteLine(fmt.Sprintf("# %s", name))
	cmake.WriteLine(fmt.Sprintf("#"))
	cmake.WriteLine("")
}

// downloadSingleHeader downloads a library consisting of a single header.
func downloadSingleHeader(cmake *cmake.CMake, headerName, guardVariable, SHA256, URL string) {
	WriteSectionComment(cmake, headerName)
	dirname := filepath.Join("${CMAKE_BINARY_DIR}", ".mkbuild", "include")
	filename := filepath.Join(dirname, headerName)
	cmake.MkdirAll(dirname)
	cmake.Download(filename, SHA256, URL)
	cmake.AddIncludeDir(dirname)
	cmake.CheckHeaderExists(headerName, guardVariable, true)
	cmake.WriteLine("")
}

// downloadWinCurl downloads curl for Windows
func downloadWinCurl(cmake *cmake.CMake, filename, SHA256, URL string) {
	dirname := filepath.Join("${CMAKE_BINARY_DIR}", ".mkbuild", "download")
	filepathname := filepath.Join(dirname, filename)
	cmake.MkdirAll(dirname)
	cmake.Download(filepathname, SHA256, URL)
	cmake.Untar(filepathname, dirname)
}

// Rules contains all the build rules that we know of.
var Rules = map[string]func(*cmake.CMake){
	"curl.haxx.se/ca": func(cmake *cmake.CMake) {
		WriteSectionComment(cmake, "ca-bundle.pem")
		dirname := filepath.Join("${CMAKE_BINARY_DIR}", ".mkbuild", "etc")
		filename := filepath.Join(dirname, "ca-bundle.pem")
		cmake.MkdirAll(dirname)
		cmake.Download(
			filename, "4d89992b90f3e177ab1d895c00e8cded6c9009bec9d56981ff4f0a59e9cc56d6",
			"https://curl.haxx.se/ca/cacert-2018-12-05.pem",
		)
	},
	"github.com/adishavit/argh": func(cmake *cmake.CMake) {
		downloadSingleHeader(cmake, "argh.h", "MK_HAVE_ARGH_H",
			"ddb7dfc18dcf90149735b76fb2cff101067453a1df1943a6911233cb7085980c",
			"https://raw.githubusercontent.com/adishavit/argh/v1.3.0/argh.h",
		)
	},
	"github.com/catchorg/catch2": func(cmake *cmake.CMake) {
		downloadSingleHeader(cmake, "catch.hpp", "MK_HAVE_CATCH_HPP",
			"5eb8532fd5ec0d28433eba8a749102fd1f98078c5ebf35ad607fb2455a000004",
			"https://github.com/catchorg/Catch2/releases/download/v2.3.0/catch.hpp",
		)
	},
	"github.com/curl/curl": func(cmake *cmake.CMake) {
		WriteSectionComment(cmake, "libcurl")
		cmake.WriteLine("if((\"${WIN32}\"))")
		cmake.WithIndent("  ", func() {
			version := "7.61.1-1"
			release := "testing"
			baseURL := "https://github.com/measurement-kit/prebuilt/releases/download/"
			URL := fmt.Sprintf("%s/%s/windows-curl-%s.tar.gz", baseURL, release, version)
			downloadWinCurl(
				cmake, "windows-curl.tar.gz",
				"424d2f18f0f74dd6a0128f0f4e59860b7d2f00c80bbf24b2702e9cac661357cf",
				URL,
			)
			cmake.WriteLine("if((\"${CMAKE_SIZEOF_VOID_P}\" EQUAL 4))")
			cmake.WithIndent("  ", func() {
				cmake.WriteLine("SET(MK_CURL_ARCH \"x86\")")
			})
			cmake.WriteLine("else()")
			cmake.WithIndent("  ", func() {
				cmake.WriteLine("SET(MK_CURL_ARCH \"x64\")")
			})
			cmake.WriteLine("endif()")
			cmake.WriteLine("")
			// E.g.: .mkbuild/download/MK_DIST/windows/curl/7.61.1-1/x86/lib/
			curldir := filepath.Join(
				"${CMAKE_BINARY_DIR}", ".mkbuild", "download", "MK_DIST",
				"windows", "curl", version, "${MK_CURL_ARCH}",
			)
			includedirname := filepath.Join(curldir, "include")
			libname := filepath.Join(curldir, "lib", "libcurl.lib")
			cmake.AddIncludeDir(includedirname)
			cmake.CheckHeaderExists("curl/curl.h", "MK_HAVE_CURL_CURL_H", true)
			cmake.WriteLine("")
			cmake.CheckLibraryExists(libname, "curl_easy_init", "MK_HAVE_LIBCURL", true)
			cmake.AddLibrary(libname)
			cmake.AddDefinition("-DCURL_STATICLIB")
		})
		cmake.WriteLine("")
		cmake.WriteLine("else()")
		cmake.WithIndent(" ", func() {
			cmake.CheckHeaderExists("curl/curl.h", "MK_HAVE_CURL_CURL_H", true)
			cmake.CheckLibraryExists("curl", "curl_easy_init", "MK_HAVE_LIBCURL", true)
			cmake.AddLibrary("curl")
		})
		cmake.WriteLine("endif()")
	},
	"github.com/measurement-kit/mkmock": func(cmake *cmake.CMake) {
		downloadSingleHeader(cmake, "mkmock.hpp", "MK_HAVE_MKMOCK_HPP",
			"f07bc063a2e64484482f986501003e45ead653ea3f53fadbdb45c17a51d916d2",
			"https://raw.githubusercontent.com/measurement-kit/mkmock/v0.2.0/mkmock.hpp",
		)
	},
}
