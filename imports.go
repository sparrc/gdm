package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build"
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
		rootpath := filepath.Join(gopath, "src", i.Repo.Root)
		err = i.Repo.VCS.CreateAtRev(rootpath, i.Repo.Repo, i.Rev)
		if err != nil {
			return fmt.Errorf("Error creating repo at %s, %s\n",
				fullpath, err.Error())
		}
		return nil
	}

	// Checkout default branch
	_, err = runInDir(i.Repo.VCS.Cmd, strings.Fields(i.Repo.VCS.TagSyncDefault),
		fullpath, i.Verbose)
	if err != nil {
		return err
	}

	// Attempt to checkout revision.
	cmdString := i.Repo.VCS.TagSyncCmd
	cmdString = strings.Replace(cmdString, "{tag}", i.Rev, 1)
	if _, err = runInDir(i.Repo.VCS.Cmd, strings.Fields(cmdString), fullpath, i.Verbose); err == nil {
		return nil
	}

	// There was an error checking out revision: download changes and
	// re-checkout revision
	err = i.Repo.VCS.Download(fullpath)
	if err != nil {
		return fmt.Errorf("Error downloading changes to %s, %s\n",
			fullpath, err.Error())
	}
	_, err = runInDir(i.Repo.VCS.Cmd, strings.Fields(cmdString), fullpath, i.Verbose)
	return err
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
	// Get a set of transitive dependencies (package import paths) for the
	// specified package.
	depsOutput, err := runInDir("go",
		[]string{"list", "-f", `{{range .Deps}}{{.}}{{"\n"}}{{end}}`, "./..."},
		wd, verbose)
	if err != nil {
		return nil, err
	}
	// filter out standard library
	deps := filterPackages(depsOutput, nil)

	// List dependencies of test files, which are not included in the go list .Deps
	// Also, ignore any dependencies that are already covered.
	testDepsOutput, err := runInDir("go",
		[]string{"list", "-f",
			`{{join .TestImports "\n"}}{{"\n"}}{{join .XTestImports "\n"}}`, "./..."},
		wd, verbose)
	if err != nil {
		return nil, err
	}
	// filter out stdlib and existing deps
	testDeps := filterPackages(testDepsOutput, deps)
	for dep := range testDeps {
		deps[dep] = true
	}

	// Sort the import set into a list of string paths
	sortedImportPaths := []string{}
	repoRoot := getImportPath(wd)
	for path, _ := range deps {
		// Do not vendor the repo that we are vendoring
		if path == repoRoot {
			continue
		}
		sortedImportPaths = append(sortedImportPaths, path)
	}
	sort.Strings(sortedImportPaths)

	// Iterate through imports, creating a list of Import structs
	result := []*Import{}
	for _, importpath := range sortedImportPaths {
		root, err := getRepoRoot(importpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting VCS info for %s, skipping\n", importpath)
			continue
		}
		_, ok := deps[root.Root]
		if root.Root == importpath || !ok {
			// Use the repo root as importpath if it's a usable go VCS repo
			if _, err := getRepoRoot(root.Root); err == nil {
				deps[root.Root] = true
				importpath = root.Root
			}
			// If this is the repo root, or root is not already imported
			fullpath := filepath.Join(gopath, "src", importpath)
			rev := getRevisionFromPath(fullpath, root)
			result = append(result, &Import{
				Rev:        rev,
				ImportPath: importpath,
				Repo:       root,
			})
		}
	}
	return result, nil
}

// getImportPath takes a path like /home/csparr/go/src/github.com/sparrc/gdm
// and returns the import path, ie, github.com/sparrc/gdm
func getImportPath(fullpath string) string {
	p, err := build.ImportDir(fullpath, 0)
	if err != nil {
		return ""
	}
	return p.ImportPath
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

// filterPackages accepts the output of a go list comment (one package per line)
// and returns a set of package import paths, excluding standard library.
// Additionally, any packages present in the "exclude" set will be excluded.
func filterPackages(output []byte, exclude map[string]bool) map[string]bool {
	var scanner = bufio.NewScanner(bytes.NewReader(output))
	var deps = map[string]bool{}
	for scanner.Scan() {
		var (
			pkg    = scanner.Text()
			slash  = strings.Index(pkg, "/")
			stdLib = slash == -1 || strings.Index(pkg[:slash], ".") == -1
		)
		if stdLib {
			continue
		}
		if _, ok := exclude[pkg]; ok {
			continue
		}
		deps[pkg] = true
	}
	return deps
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
