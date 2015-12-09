package main

import (
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

var (
	// A set of the imports, gets modified by tree walker
	importSet map[string]bool

	// rootImport is the import path of the root repo, ie, working directory.
	rootImport string
)

type Import struct {
	// ie, golang.org/x/tools/go/vcs
	ImportPath string

	// ie, 759e96ebaffb01c3cba0e8b129ef29f56507b323
	Rev string

	Repo *vcs.RepoRoot
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

func ImportsFromPath(wd, gopath string) []*Import {
	// Get all imports from root directory
	setRootImport(wd)
	filepath.Walk(wd, visit)

	// Sort the import set into a list of string paths
	sortedImportSet := []string{}
	for path, _ := range importSet {
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
		_, ok := importSet[root.Root]
		if root.Root == path || !ok {
			// Use the repo root as path if it's a usable go VCS repo
			if _, err := getRepoRoot(root.Root); err == nil {
				importSet[root.Root] = true
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

func setRootImport(path string) {
	p, err := build.ImportDir(path, 0)
	if err != nil {
		return
	}
	rootImport = p.ImportPath
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

func visit(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		return nil
	}
	p, err := build.ImportDir(path, 0)
	if err != nil {
		return nil
	}

	for _, i := range p.Imports {
		if strings.Contains(i, rootImport) {
			continue
		} else if strings.Contains(i, ".") && strings.Contains(i, "/") {
			importSet[i] = true
		}
	}

	for _, i := range p.TestImports {
		if strings.Contains(i, rootImport) {
			continue
		} else if strings.Contains(i, ".") && strings.Contains(i, "/") {
			importSet[i] = true
		}
	}
	return nil
}

func init() {
	importSet = make(map[string]bool)
}
