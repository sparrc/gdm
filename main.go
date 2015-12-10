package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// Version can be auto-set at build time using an ldflag
	//   go build -ldflags "-X main.Version `git describe --tags --always`"
	Version string

	// DepsFile specifies the Godeps file used by gdm
	DepsFile string = "Godeps"
)

const usage = `Go Dependency Manager (gdm), a lightweight tool for managing Go dependencies.

Usage:

    gdm <command> [-f GODEPS_FILE] [-v]

The commands are:

    restore   Check out revisions defined in Godeps file to $GOPATH.
    save      Saves currently checked-out dependencies from $GOPATH to Godeps file.
`

func main() {
	flag.Usage = usageExit
	flag.Parse()
	args := flag.Args()
	var ffile string
	var verbose bool
	if len(args) < 1 {
		usageExit()
	} else if len(args) > 1 {
		set := flag.NewFlagSet("", flag.ExitOnError)
		set.StringVar(&ffile, "f", "Godeps", "Specify the name/location of Godeps file")
		set.BoolVar(&verbose, "v", false, "Verbose mode")
		set.Parse(os.Args[2:])
	}
	if ffile != "" {
		DepsFile = ffile
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("======= Go Dependency Manager =======")
	fmt.Println("= working dir:", wd)
	gopath := getGoPath(wd)
	fmt.Println("= GOPATH:     ", gopath)
	fmt.Println("=====================================")

	switch args[0] {
	case "save", "bootstrap":
		save(wd, gopath, verbose)
	case "restore", "get", "sync", "checkout":
		restore(wd, gopath, verbose)
	}
}

func usageExit() {
	fmt.Println(usage)
	os.Exit(0)
}

func getGoPath(wd string) string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		fmt.Fprintf(os.Stderr, "GOPATH must be set to use gdm")
		os.Exit(1)
	}

	// Split out multiple GOPATHs if necessary
	if strings.Contains(gopath, ":") {
		paths := strings.Split(gopath, ":")
		for _, path := range paths {
			if strings.Contains(path, wd) {
				gopath = path
				break
			}
		}
	}

	if !strings.Contains(wd, gopath) {
		fmt.Fprintf(os.Stderr, "gdm can only be executed within a directory in the GOPATH")
		os.Exit(1)
	}
	return gopath
}

func save(wd, gopath string, verbose bool) {
	imports := ImportsFromPath(wd, gopath, verbose)

	f, err := os.Create(filepath.Join(wd, DepsFile))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, i := range imports {
		fmt.Printf("> Saving Import [%s] Revision [%s]\n", i.ImportPath, i.Rev)
		_, err = w.WriteString(fmt.Sprintf("%s %s\n", i.ImportPath, i.Rev))
		if err != nil {
			panic(err)
		}
	}
	w.Flush()
}

func restore(wd, gopath string, verbose bool) {
	imports := ImportsFromFile(filepath.Join(wd, DepsFile))
	for _, i := range imports {
		i.Verbose = verbose
		i.RestoreImport(gopath)
	}
}
