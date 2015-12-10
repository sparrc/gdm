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

type Import struct {
	// ie, golang.org/x/tools/go/vcs
	ImportPath string

	// ie, 759e96ebaffb01c3cba0e8b129ef29f56507b323
	Rev string

	// Controls verbosity of output
	Verbose bool

	Repo *vcs.RepoRoot
}

func (i *Import) RestoreImport(gopath string) {
	vcs.ShowCmd = i.Verbose
	fullpath := filepath.Join(gopath, "src", i.ImportPath)
	fmt.Printf("> Restoring %s to %s\n", fullpath, i.Rev)

	// Checkout default branch
	runInDir(i.Repo.VCS.Cmd, strings.Fields(i.Repo.VCS.TagSyncDefault),
		fullpath, i.Verbose)

	// Download changes
	err := i.Repo.VCS.Download(fullpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading changes to %s, %s\n",
			fullpath, err.Error())
		os.Exit(1)
	}

	// Checkout revision
	cmdString := i.Repo.VCS.TagSyncCmd
	cmdString = strings.Replace(cmdString, "{tag}", i.Rev, 1)
	runInDir(i.Repo.VCS.Cmd, strings.Fields(cmdString),
		fullpath, i.Verbose)
}

func ImportsFromFile(filename string) []*Import {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")
	imports := []*Import{}
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
		imports = append(imports, &Import{
			Rev:        rev,
			ImportPath: path,
			Repo:       root,
		})
	}
	return imports
}

func ImportsFromPath(wd, gopath string, verbose bool) []*Import {
	// Get a set of transitive dependencies (package import paths) for the
	// specified package.
	depsOutput := runInDir("go",
		[]string{"list", "-f", `{{range .Deps}}{{.}}{{"\n"}}{{end}}`, "./..."},
		wd, verbose)
	// filter out standard library
	deps := filterPackages(depsOutput, nil)

	// List dependencies of test files, which are not included in the go list .Deps
	// Also, ignore any dependencies that are already covered.
	testDepsOutput := runInDir("go",
		[]string{"list", "-f",
			`{{join .TestImports "\n"}}{{"\n"}}{{join .XTestImports "\n"}}`, "./..."},
		wd, verbose)
	// filter out stdlib and existing deps
	testDeps := filterPackages(testDepsOutput, deps)
	for dep := range testDeps {
		deps[dep] = true
	}

	// Sort the import set into a list of string paths
	sortedImportSet := []string{}
	repoRoot := getRootImport(wd)
	for path, _ := range deps {
		// Do not vendor the repo that we are vendoring
		if path == repoRoot {
			continue
		}
		sortedImportSet = append(sortedImportSet, path)
	}
	sort.Strings(sortedImportSet)

	// Iterate through imports, creating a list of Import structs
	imports := []*Import{}
	for _, path := range sortedImportSet {
		root, err := getRepoRoot(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting VCS info for %s, skipping\n", path)
			continue
		}
		_, ok := deps[root.Root]
		if root.Root == path || !ok {
			// Use the repo root as path if it's a usable go VCS repo
			if _, err := getRepoRoot(root.Root); err == nil {
				deps[root.Root] = true
				path = root.Root
			}
			// If this is the repo root, or root is not already imported
			fullpath := filepath.Join(gopath, "src", path)
			rev := getRevisionFromPath(fullpath, root)
			imports = append(imports, &Import{
				Rev:        rev,
				ImportPath: path,
				Repo:       root,
			})
		}
	}
	return imports
}

// getRepoImport takes a path like /home/csparr/go/src/github.com/sparrc/gdm
// and returns the import path, ie, github.com/sparrc/gdm
func getRootImport(path string) string {
	p, err := build.ImportDir(path, 0)
	if err != nil {
		return ""
	}
	return p.ImportPath
}

func getRepoRoot(path string) (*vcs.RepoRoot, error) {
	repo, err := vcs.RepoRootForImportPath(path, false)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

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

func runInDir(name string, args []string, dir string, verbose bool) []byte {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if verbose {
		fmt.Printf("cd %s\n%s %s\n", dir, name, strings.Join(args, " "))
	}
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s %s in dir %s, %s\n",
			name, strings.Join(args, " "), dir, err.Error())
		os.Exit(1)
	}
	return output
}
