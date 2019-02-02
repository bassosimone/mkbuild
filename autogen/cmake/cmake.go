// Package cmake implements the CMake driver
package cmake

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/bassosimone/mkbuild/autogen/cmake/restrictiveflags"
	"github.com/bassosimone/mkbuild/autogen/prebuilt"
)

// CMake is the CMake driver
type CMake struct {
	// output contains the CMakeLists.txt lines
	output strings.Builder

	// indent is the indent string to prefix to each line
	indent string
}

// withIndent runs |func| with the specified |indent|.
func (cmake *CMake) withIndent(indent string, fn func()) {
	oldIndent := cmake.indent
	cmake.indent += indent
	fn()
	cmake.indent = oldIndent
}

// writeSectionComment writes a comment for |name| in |cmake|.
func (cmake *CMake) writeSectionComment(name string) {
	cmake.writeEmptyLine()
	cmake.writeLine(fmt.Sprintf("#"))
	cmake.writeLine(fmt.Sprintf("# %s", name))
	cmake.writeLine(fmt.Sprintf("#"))
	cmake.writeEmptyLine()
}

// writeEmptyLine writes an empty line to output.
func (cmake *CMake) writeEmptyLine() {
	cmake.writeLine("")
}

// writeLine writes a line to the CMakeLists.txt file.
func (cmake *CMake) writeLine(s string) {
	if s != "" {
		_, err := cmake.output.WriteString(cmake.indent)
		if err != nil {
			log.WithError(err).Fatal("cannot write indent")
		}
		_, err = cmake.output.WriteString(s)
		if err != nil {
			log.WithError(err).Fatal("cannot write string")
		}
	}
	_, err := cmake.output.WriteString("\n")
	if err != nil {
		log.WithError(err).Fatal("cannot write newline")
	}
}

// Open opens a CMake project named |name|.
func Open(name string) *CMake {
	cmake := &CMake{}
	cmake.writeLine("# Autogenerated file; DO NOT EDIT!")
	cmake.writeLine(fmt.Sprintf("cmake_minimum_required(VERSION 3.12.0)"))
	cmake.writeLine(fmt.Sprintf("project(\"%s\")", name))
	cmake.writeEmptyLine()
	cmake.writeLine("include(CheckIncludeFileCXX)")
	cmake.writeLine("include(CheckLibraryExists)")
	cmake.writeLine("include(CheckCXXCompilerFlag)")
	cmake.writeLine("set(THREADS_PREFER_PTHREAD_FLAG ON)")
	cmake.writeLine("find_package(Threads REQUIRED)")
	cmake.writeLine("set(CMAKE_POSITION_INDEPENDENT_CODE ON)")
	cmake.writeLine("set(CMAKE_CXX_STANDARD 11)")
	cmake.writeLine("set(CMAKE_CXX_STANDARD_REQUIRED ON)")
	cmake.writeLine("set(CMAKE_CXX_EXTENSIONS OFF)")
	cmake.writeLine("set(CMAKE_C_STANDARD 11)")
	cmake.writeLine("set(CMAKE_C_STANDARD_REQUIRED ON)")
	cmake.writeLine("set(CMAKE_C_EXTENSIONS OFF)")
	cmake.writeLine("list(APPEND CMAKE_REQUIRED_LIBRARIES Threads::Threads)")
	cmake.writeLine("if(\"${WIN32}\")")
	cmake.writeLine("  list(APPEND CMAKE_REQUIRED_LIBRARIES ws2_32 crypt32)")
	cmake.writeLine("  if(\"${MINGW}\")")
	cmake.writeLine("    list(APPEND CMAKE_REQUIRED_LIBRARIES -static-libgcc -static-libstdc++)")
	cmake.writeLine("  endif()")
	cmake.writeLine("endif()")
	cmake.writeEmptyLine()
	cmake.writeLine("enable_testing()")
	cmake.writeEmptyLine()
	cmake.if32bit(func() {
		cmake.writeLine("SET(MK_ARCH \"x86\")")
	}, func() {
		cmake.writeLine("SET(MK_ARCH \"x64\")")
	})
	return cmake
}

// download downloads |URL| to |filename| and checks the |SHA256|.
func (cmake *CMake) download(filename, SHA256, URL string) {
	cmake.writeLine(fmt.Sprintf("message(STATUS \"download: %s\")", URL))
	cmake.writeLine(fmt.Sprintf("file(DOWNLOAD %s", URL))
	cmake.writeLine(fmt.Sprintf("  \"%s\"", filename))
	cmake.writeLine(fmt.Sprintf("  EXPECTED_HASH SHA256=%s", SHA256))
	cmake.writeLine(fmt.Sprintf("  TLS_VERIFY ON)"))
}

// checkCommandError writes the code to check for errors after a
// command has been executed.
func (cmake *CMake) checkCommandError() {
	cmake.writeLine(fmt.Sprintf("if(\"${FAILURE}\")"))
	cmake.writeLine(fmt.Sprintf("  message(FATAL_ERROR \"${FAILURE}\")"))
	cmake.writeLine(fmt.Sprintf("endif()"))
}

// MkdirAll creates |destdirs|.
func (cmake *CMake) MkdirAll(destdirs string) {
	cmake.writeLine(fmt.Sprintf("message(STATUS \"MkdirAll: %s\")", destdirs))
	cmake.writeLine(fmt.Sprintf("execute_process(COMMAND"))
	cmake.writeLine(fmt.Sprintf(
		"  ${CMAKE_COMMAND} -E make_directory \"%s\"", destdirs,
	))
	cmake.writeLine(fmt.Sprintf("  RESULT_VARIABLE FAILURE)"))
	cmake.checkCommandError()
}

// Unzip extracts |filename| in |destdir|.
func (cmake *CMake) Unzip(filename, destdir string) {
	cmake.writeLine(fmt.Sprintf("message(STATUS \"Extract: %s\")", filename))
	cmake.writeLine(fmt.Sprintf("execute_process(COMMAND"))
	cmake.writeLine(fmt.Sprintf(
		"  ${CMAKE_COMMAND} -E tar xf \"%s\"", filename,
	))
	cmake.writeLine(fmt.Sprintf("  WORKING_DIRECTORY \"%s\"", destdir))
	cmake.writeLine(fmt.Sprintf("  RESULT_VARIABLE FAILURE)"))
	cmake.checkCommandError()
}

// Untar extracts |filename| in |destdir|.
func (cmake *CMake) Untar(filename, destdir string) {
	cmake.Unzip(filename, destdir)
}

// Copy copies source to dest.
func (cmake *CMake) Copy(source, dest string) {
	cmake.writeLine(fmt.Sprintf("message(STATUS \"Copy: %s %s\")", source, dest))
	cmake.writeLine(fmt.Sprintf("execute_process(COMMAND"))
	cmake.writeLine(fmt.Sprintf(
		"  ${CMAKE_COMMAND} -E copy \"%s\" \"%s\"", source, dest,
	))
	cmake.writeLine(fmt.Sprintf("  RESULT_VARIABLE FAILURE)"))
	cmake.checkCommandError()
}

// CopyDir copies source to dest.
func (cmake *CMake) CopyDir(source, dest string) {
	cmake.writeLine(fmt.Sprintf(
		"message(STATUS \"CopyDir: %s %s\")", source, dest,
	))
	cmake.writeLine(fmt.Sprintf("execute_process(COMMAND"))
	cmake.writeLine(fmt.Sprintf(
		"  ${CMAKE_COMMAND} -E copy_directory \"%s\" \"%s\"", source, dest,
	))
	cmake.writeLine(fmt.Sprintf("  RESULT_VARIABLE FAILURE)"))
	cmake.checkCommandError()
}

// AddDefinition adds |definition| to the macro definitions
func (cmake *CMake) AddDefinition(definition string) {
	cmake.writeLine(fmt.Sprintf(
		"LIST(APPEND CMAKE_REQUIRED_DEFINITIONS %s)", definition,
	))
}

// AddIncludeDir adds |path| to the header search path
func (cmake *CMake) AddIncludeDir(path string) {
	cmake.writeLine(fmt.Sprintf(
		"LIST(APPEND CMAKE_REQUIRED_INCLUDES \"%s\")", path,
	))
}

// AddLibrary adds |library| to the libraries to include
func (cmake *CMake) AddLibrary(library string) {
	cmake.writeLine(fmt.Sprintf(
		"LIST(APPEND CMAKE_REQUIRED_LIBRARIES \"%s\")", library,
	))
}

// checkPlatformCheckResult writes code to deal with a platform check result.
func (cmake *CMake) checkPlatformCheckResult(item, variable string, mandatory bool) {
	if mandatory {
		cmake.writeLine(fmt.Sprintf("if(NOT (\"${%s}\"))", variable))
		cmake.writeLine(fmt.Sprintf(
			"  message(FATAL_ERROR \"cannot find: %s\")", item,
		))
		cmake.writeLine(fmt.Sprintf("endif()"))
	}
}

// CheckHeaderExists checks whether |header| exists and stores the
// result into the specified |variable|. If |mandatory| then, the
// processing will stop on failure. Otherwise, if found, then we'll
// add a preprocessor symbol named after |variable|.
func (cmake *CMake) CheckHeaderExists(header, variable string, mandatory bool) {
	cmake.writeLine(fmt.Sprintf(
		"CHECK_INCLUDE_FILE_CXX(\"%s\" %s)", header, variable,
	))
	cmake.checkPlatformCheckResult(header, variable, mandatory)
}

// CheckLibraryExists checks whether |library| exists by looking for
// a function named |function|, storing the result in |variable|.
func (cmake *CMake) CheckLibraryExists(library, function, variable string, mandatory bool) {
	cmake.writeLine(fmt.Sprintf(
		"CHECK_LIBRARY_EXISTS(\"%s\" \"%s\" \"\" %s)", library, function, variable,
	))
	cmake.checkPlatformCheckResult(library, variable, mandatory)
}

// setRestrictiveCompilerFlags sets restrictive compiler flags.
func (cmake *CMake) setRestrictiveCompilerFlags() {
	cmake.writeSectionComment("Set restrictive compiler flags")
	cmake.output.WriteString(restrictiveflags.S)
	cmake.writeEmptyLine()
	cmake.writeLine(fmt.Sprintf("MkSetCompilerFlags()"))
}

// prepareForCompilingTargets prepares internal variables such that
// we can compile targets with the required compiler flags.
func (cmake *CMake) prepareForCompilingTargets() {
	cmake.writeSectionComment("Prepare for compiling targets")
	cmake.writeLine("add_definitions(${CMAKE_REQUIRED_DEFINITIONS})")
	cmake.writeLine("include_directories(${CMAKE_REQUIRED_INCLUDES})")
}

// BuildExecutable defines an executable to be compiled.
func (cmake *CMake) BuildExecutable(name string, sources []string, libs []string) {
	cmake.writeSectionComment(name)
	cmake.writeLine(fmt.Sprintf("add_executable("))
	cmake.writeLine(fmt.Sprintf("  %s", name))
	for _, source := range sources {
		cmake.writeLine(fmt.Sprintf("  %s", source))
	}
	cmake.writeLine(fmt.Sprintf(")"))
	cmake.writeLine(fmt.Sprintf("target_link_libraries("))
	cmake.writeLine(fmt.Sprintf("  %s", name))
	for _, lib := range libs {
		cmake.writeLine(fmt.Sprintf("  %s", lib))
	}
	cmake.writeLine(fmt.Sprintf("  ${CMAKE_REQUIRED_LIBRARIES}"))
	cmake.writeLine(fmt.Sprintf(")"))
}

// BuildLibrary defines a static library to be compiled.
func (cmake *CMake) BuildLibrary(name string, sources []string, libs []string) {
	cmake.writeSectionComment(name)
	cmake.writeLine(fmt.Sprintf("add_library("))
	cmake.writeLine(fmt.Sprintf("  %s", name))
	for _, source := range sources {
		cmake.writeLine(fmt.Sprintf("  %s", source))
	}
	cmake.writeLine(fmt.Sprintf(")"))
	cmake.writeLine(fmt.Sprintf("target_link_libraries("))
	cmake.writeLine(fmt.Sprintf("  %s", name))
	for _, lib := range libs {
		cmake.writeLine(fmt.Sprintf("  %s", lib))
	}
	cmake.writeLine(fmt.Sprintf("  ${CMAKE_REQUIRED_LIBRARIES}"))
	cmake.writeLine(fmt.Sprintf(")"))
}

// RunTest defines a test to be run
func (cmake *CMake) RunTest(name, command string) {
	cmake.writeSectionComment("test: " + name)
	cmake.writeLine(fmt.Sprintf("add_test("))
	cmake.writeLine(fmt.Sprintf("  NAME %s COMMAND %s", name, command))
	cmake.writeLine(fmt.Sprintf(")"))
}

// AddSingleHeaderDependency adds a single-header dependency
func (cmake *CMake) AddSingleHeaderDependency(SHA256, URL string) {
	headerName := filepath.Base(URL)
	cmake.writeSectionComment(headerName)
	dirname := "${CMAKE_BINARY_DIR}/.mkbuild/include"
	filename := dirname + "/" + headerName
	cmake.MkdirAll(dirname)
	cmake.download(filename, SHA256, URL)
	cmake.AddIncludeDir(dirname)
	guardVariable := "MK_HAVE_" + strings.ToUpper(strings.Replace(headerName, ".", "_", -1))
	cmake.CheckHeaderExists(headerName, guardVariable, true)
}

// AddSingleFileAsset adds a single-file asset to the build
func (cmake *CMake) AddSingleFileAsset(SHA256, URL string) {
	assetName := filepath.Base(URL)
	cmake.writeSectionComment(assetName)
	dirname := "${CMAKE_BINARY_DIR}/.mkbuild/data"
	filename := dirname + "/" + assetName
	cmake.MkdirAll(dirname)
	cmake.download(filename, SHA256, URL)
}

// IfWIN32 allows you to generate WIN32 / !WIN32 specific code.
func (cmake *CMake) IfWIN32(thenFunc func(), elseFunc func()) {
	cmake.writeLine("if((\"${WIN32}\"))")
	cmake.withIndent("  ", thenFunc)
	cmake.writeLine("else()")
	cmake.withIndent("  ", elseFunc)
	cmake.writeLine("endif()")
}

// if32bit allows you to generate 32 bit / 64 bit specific code. This
// function will configure cmake to fail if the bitsize is neither
// 32 not 64. That would be a very weird configuraton.
func (cmake *CMake) if32bit(func32 func(), func64 func()) {
	cmake.writeLine("if((\"${CMAKE_SIZEOF_VOID_P}\" EQUAL 4))")
	cmake.withIndent("  ", func32)
	cmake.writeLine("elseif((\"${CMAKE_SIZEOF_VOID_P}\" EQUAL 8))")
	cmake.withIndent("  ", func64)
	cmake.writeLine("else()")
	cmake.withIndent("  ", func() {
		cmake.writeLine("message(FATAL_ERROR \"Neither 32 not 64 bit\")")
	})
	cmake.writeLine("endif()")
}

// Win32InstallPrebuilt installs a prebuilt Windows package.
func (cmake *CMake) Win32InstallPrebuilt(info *prebuilt.Info) {
	cmake.downloadAndExtractArchive(info.SHA256, info.URL)
	basedir := "${CMAKE_BINARY_DIR}/.mkbuild/download/" + info.Prefix + "/${MK_ARCH}"
	includedirname := basedir + "/include"
	libnameFull := basedir + "/lib/" + info.LibName
	cmake.AddIncludeDir(includedirname)
	cmake.CheckHeaderExists(info.HeaderName, "MK_WIN32_HAVE_HEADER", true)
	cmake.CheckLibraryExists(libnameFull, info.FuncName, "MK_WIN32_HAVE_LIBRARY", true)
	cmake.AddLibrary(libnameFull)
}

// downloadAndExtractArchive downloads and extracts and archive
func (cmake *CMake) downloadAndExtractArchive(SHA256, URL string) {
	archiveName := filepath.Base(URL)
	cmake.writeSectionComment(archiveName)
	dirname := "${CMAKE_BINARY_DIR}/.mkbuild/download"
	filename := dirname + "/" + archiveName
	cmake.MkdirAll(dirname)
	cmake.download(filename, SHA256, URL)
	filepathname := dirname + "/" + archiveName
	cmake.Untar(filepathname, dirname)
}

// FinalizeCompilerFlags finalizes compiler flags
func (cmake *CMake) FinalizeCompilerFlags() {
	cmake.setRestrictiveCompilerFlags()
	cmake.prepareForCompilingTargets()
}

// Close writes CMakeLists.txt in the current directory.
func (cmake *CMake) Close() {
	filename := "CMakeLists.txt"
	filep, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.WithError(err).Fatalf("os.Open failed for: %s", filename)
	}
	defer filep.Close()
	_, err = filep.WriteString(cmake.output.String())
	if err != nil {
		log.WithError(err).Fatalf("filep.WriteString failed for: %s", filename)
	}
	log.Infof("Written %s", filename)
}
