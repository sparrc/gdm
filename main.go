package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// Version can be auto-set at build time using an ldflag
	//   go build -ldflags "-X main.Version `git describe --tags --always`"
	Version string
)

const usage = `Go Dependency Manager (gdm), a lightweight tool for managing Go dependencies.

Usage:

    gdm <command>

The commands are:

    restore   Check out revisions defined in Godeps file in GOPATH.
    save      Saves currently checked-out dependencies from GOPATH to Godeps file.
`

const DepsFile string = "Godeps"

func main() {
	flag.Usage = usageExit
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usageExit()
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
		save(wd, gopath)
	case "restore", "get", "sync", "checkout":
		restore(wd, gopath)
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

func save(wd, gopath string) {
	imports := ImportsFromPath(wd, gopath)

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

func restore(wd, gopath string) {
	imports := ImportsFromFile(filepath.Join(wd, DepsFile))
	for _, i := range imports {
		checkoutRevision(i, gopath)
	}
}

func checkoutRevision(i *Import, gopath string) {
	fullpath := filepath.Join(gopath, "src", i.ImportPath)
	cmdString := i.Repo.VCS.TagSyncCmd
	cmdString = strings.Replace(cmdString, "{tag}", i.Rev, 1)
	cmd := exec.Command(i.Repo.VCS.Cmd, strings.Split(cmdString, " ")...)
	cmd.Dir = fullpath
	fmt.Printf("> Executing [%s %s] in %s\n", i.Repo.VCS.Cmd, cmdString, fullpath)
	_, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking out revision at %s, %s\n",
			fullpath, err.Error())
		os.Exit(1)
	}
}
