package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/vcs"
)

// Import defines an import dependency
type Import struct {
	// ie, golang.org/x/tools/go/vcs
	ImportPath string

	// ie, 759e96ebaffb01c3cba0e8b129ef29f56507b323
	Rev string

	// Controls verbosity of output
	Verbose bool

	// see https://godoc.org/golang.org/x/tools/go/vcs#RepoRoot
	Repo *vcs.RepoRoot
}

// RestoreImport takes the import and restores it at the given GOPATH.
// There are four steps to this:
//   1. cd $GOPATH/src/<import_path>
//   2. Checkout default branch (ie, git checkout master)
//   3. Download changes (ie, git pull --ff-only)
//   4. Checkout revision (ie, git checkout 759e96ebaffb01c3cba0e8b129ef29f56507b323)
func (i *Import) RestoreImport(gopath string) error {
	vcs.ShowCmd = i.Verbose
	fullpath := filepath.Join(gopath, "src", i.ImportPath)
	fmt.Printf("> Restoring %s to %s\n", fullpath, i.Rev)

	// If the repo doesn't exist already, create it
	_, err := os.Stat(fullpath)
	if err != nil && os.IsNotExist(err) {
		if i.Verbose {
			fmt.Printf("> Repo %s not found, creating at rev %s\n", fullpath, i.Rev)
		}

		// Create parent directory
		rootpath := filepath.Join(gopath, "src", i.Repo.Root)
		if err = os.MkdirAll(rootpath, os.ModePerm); err != nil {
			return fmt.Errorf("Could not create parent directory %s for repo %s\n",
				rootpath, fullpath)
		}

		// Clone repo
		if err = i.Repo.VCS.Create(rootpath, i.Repo.Repo); err != nil {
			return fmt.Errorf("Error cloning repo at %s, %s\n",
				fullpath, err.Error())
		}
	}

	// Attempt to checkout revision.
	cmdString := i.Repo.VCS.TagSyncCmd
	cmdString = strings.Replace(cmdString, "{tag}", i.Rev, 1)
	if _, err = runInDir(i.Repo.VCS.Cmd, strings.Fields(cmdString), fullpath, i.Verbose); err == nil {
		return nil
	}

	// Revision not found, checkout default branch (usually master).
	_, err = runInDir(i.Repo.VCS.Cmd, strings.Fields(i.Repo.VCS.TagSyncDefault),
		fullpath, i.Verbose)
	if err != nil {
		return fmt.Errorf("Error checking out default branch (usually master) in repo %s, %s\n",
			fullpath, err.Error())
	}

	// Download changes from remote repo.
	err = i.Repo.VCS.Download(fullpath)
	if err != nil {
		return fmt.Errorf("Error downloading changes to %s, %s\n",
			fullpath, err.Error())
	}

	// Attempt to checkout rev again after downloading changes.
	if _, err = runInDir(i.Repo.VCS.Cmd, strings.Fields(cmdString), fullpath, i.Verbose); err != nil {
		return fmt.Errorf("Error checking out rev %s of repo %s, %s\n",
			i.Rev, fullpath, err.Error())
	}
	return nil
}

// CheckImport checks if the import is at the proper revision.
func (i *Import) CheckImport(gopath string) (bool, error) {
	fullpath := filepath.Join(gopath, "src", i.ImportPath)
	if _, err := os.Stat(fullpath); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return false, err
	}

	// Retrieve the current revision.
	var args []string
	switch i.Repo.VCS.Cmd {
	case "git":
		args = []string{"rev-parse", "HEAD"}
	case "hg":
		args = []string{"log", "-l1", "--template", "{node}"}
	default:
		return false, nil
	}

	out, err := runInDir(i.Repo.VCS.Cmd, args, fullpath, i.Verbose)
	if err != nil {
		return false, err
	}
	rev := strings.TrimSpace(string(out))
	if i.Verbose {
		fmt.Printf("> Comparing repo %s revision %s to %s\n", fullpath, rev, i.Rev)
	}
	return rev == i.Rev, nil
}

// ImportsFromFile reads the given file and returns Import structs.
func ImportsFromFile(filename string) []*Import {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")
	imports := []*Import{}
	roots := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || len(line) == 0 {
			// Skip commented line
			continue
		} else if strings.Contains(line, "#") {
			// in-line comment
			line = strings.TrimSpace(strings.Split(line, "#")[0])
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid line: %s\n", line)
			os.Exit(1)
		}
		path := parts[0]
		rev := parts[1]
		root, err := getRepoRoot(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting VCS info for %s\n", path)
			os.Exit(1)
		}
		if _, ok := roots[root.Root]; !ok {
			roots[root.Root] = true
			imports = append(imports, &Import{
				Rev:        rev,
				ImportPath: path,
				Repo:       root,
			})
		}
	}
	return imports
}

// ImportsFromPath looks in the given working directory and finds all 3rd-party
// imports, and returns Import structs
func ImportsFromPath(wd, gopath string, verbose bool) ([]*Import, error) {
	// Parse the files within the working directory.
	pkgs, err := ParseImports(filepath.Join(wd, "..."))
	if err != nil {
		return nil, err
	}

	// Retrieve a mapping of every repository from the import paths.
	deps := make(map[string]*Import)
	for _, pkg := range pkgs {
		// Ignore packages founds on the GOROOT.
		if pkg.Goroot {
			continue
		}

		proot, err := getRepoRoot(pkg.ImportPath)
		if err != nil {
			return nil, err
		}

		// Check if this repository has been referenced by another import.
		if _, ok := deps[proot.Root]; ok {
			continue
		}

		// Create the Import object.
		fullpath := filepath.Join(pkg.SrcRoot, proot.Root)
		rev := getRevisionFromPath(fullpath, proot)
		deps[proot.Root] = &Import{
			Rev:        rev,
			ImportPath: proot.Root,
			Repo:       proot,
		}
	}

	sortedImportPaths := make([]string, 0, len(deps))
	for importpath := range deps {
		sortedImportPaths = append(sortedImportPaths, importpath)
	}
	sort.Strings(sortedImportPaths)

	// Iterate through imports creating a slice of Imports.
	result := []*Import{}
	for _, importpath := range sortedImportPaths {
		result = append(result, deps[importpath])
	}
	return result, nil
}

// getRepoRoot takes an import path like github.com/sparrc/gdm
// and returns the VCS Repository information for it.
func getRepoRoot(importpath string) (*vcs.RepoRoot, error) {
	repo, err := vcs.RepoRootForImportPath(importpath, false)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// getRevisionFromPath takes a path like /home/csparr/go/src/github.com/sparrc/gdm
// and the VCS Repository information and returns the currently checked out
// revision, ie, 759e96ebaffb01c3cba0e8b129ef29f56507b323
func getRevisionFromPath(fullpath string, root *vcs.RepoRoot) string {
	// Check that we have the executable available
	_, err := exec.LookPath(root.VCS.Cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gdm missing %s command.\n", root.VCS.Name)
		os.Exit(1)
	}

	// Determine command to get the current hash
	var cmd *exec.Cmd
	switch root.VCS.Cmd {
	case "git":
		cmd = exec.Command("git", "rev-parse", "HEAD")
	case "hg":
		cmd = exec.Command("hg", "id", "-i")
	case "bzr":
		cmd = exec.Command("bzr", "revno")
	default:
		fmt.Fprintf(os.Stderr, "gdm does not support %s\n", root.VCS.Cmd)
		os.Exit(1)
	}
	cmd.Dir = fullpath

	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting revision hash at %s, %s\n",
			fullpath, err.Error())
		os.Exit(1)
	}
	return strings.TrimSpace(string(output))
}

// runInDir runs the given command (name) with args, in the given directory.
// if verbose, prints out the command and dir it is executing.
// This function exits the whole program if it fails.
// Returns output of the command.
func runInDir(name string, args []string, dir string, verbose bool) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if verbose {
		fmt.Printf("cd %s\n%s %s\n", dir, name, strings.Join(args, " "))
	}
	output, err := cmd.Output()
	if err != nil {
		fmt.Errorf("Error running %s %s in dir %s, %s\n",
			name, strings.Join(args, " "), dir, err.Error())
		return output, err
	}
	return output, nil
}
