package main

import (
	"bytes"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func vendor(wd, gopath string, verbose bool) {
	// Load the Godeps file where we will retrieve our dependencies from.
	imports := ImportsFromFile(filepath.Join(wd, DepsFile))
	if len(imports) == 0 {
		fmt.Println("Fatal error: Unable to vendor imports (no Godeps file)")
		os.Exit(1)
	}

	// Read the desired imports from our source files.
	// We want to get the smallest subset of folders rather than copying the entire
	// dependency so we reparse the imports.
	pkgs, err := ParseImports("./...")
	if err != nil {
		fmt.Printf("Fatal error: %s\n", err)
		os.Exit(1)
	}

	// Map each of these import paths to an import in the Godeps file and vendor it.
	for _, pkg := range pkgs {
		// Ignore vendoring packages found on the GOROOT.
		if pkg.Goroot {
			continue
		}

		// Retrieve the import for this package.
		var I *Import
		for _, i := range imports {
			if strings.HasPrefix(pkg.ImportPath, i.ImportPath) {
				I = i
				break
			}
		}

		// If no suitable import could be found, exit with an error.
		if I == nil {
			fmt.Printf("Fatal error: Unknown import at path %s. Run gdm save.\n", pkg.ImportPath)
			os.Exit(1)
		}

		// Attempt to restore the import just in case it was modified.
		if ok, err := I.CheckImport(pkg.Root); err != nil {
			fmt.Printf("Fatal error: %s\n", err)
			os.Exit(1)
		} else if !ok {
			if err := I.RestoreImport(pkg.Root); err != nil {
				fmt.Printf("Fatal error: %s\n", err)
				os.Exit(1)
			}
		}

		// Vendor the path.
		vendorDir := filepath.Join(wd, "vendor", pkg.ImportPath)
		fmt.Printf("> Vendoring %s to %s\n", pkg.Dir, vendorDir)

		// Create import path in the vendor directory.
		if err := os.MkdirAll(vendorDir, os.ModePerm); err != nil {
			fmt.Printf("Fatal error: %s\n", err)
			os.Exit(1)
		}

		// Rsync the directory excluding any source control directories.
		if err := copyDir(vendorDir, pkg.Dir); err != nil {
			fmt.Printf("Fatal error: %s\n", err)
			os.Exit(1)
		}
	}
}

func copyDir(dest, src string) error {
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}

	var buf bytes.Buffer
	cmd := exec.Command("rsync", "-a", "--exclude=.git/", "--exclude=.hg/", "--exclude=.svn/", src+"/", dest+"/")
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		parts := strings.SplitN(buf.String(), "\n", 2)
		return fmt.Errorf("unable to copy %s to %s: %s", src, dest, parts[0])
	}
	return nil
}

func ExpandPaths(paths ...string) ([]string, error) {
	a := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(path, "/...") {
			dir := path[:len(path)-4]
			var walkDir func(dir string) error
			walkDir = func(dir string) error {
				dirs, err := ioutil.ReadDir(dir)
				if err != nil {
					return err
				}

				for _, d := range dirs {
					if !d.IsDir() || strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" {
						continue
					}

					path := dir + string(os.PathSeparator) + d.Name()
					a = append(a, path)
					if err := walkDir(path); err != nil {
						return err
					}
				}
				return nil
			}

			a = append(a, dir)
			if err := walkDir(dir); err != nil {
				return nil, err
			}
			continue
		}
		a = append(a, path)
	}
	return a, nil
}

type dependency struct {
	Name string
	Dir  string
}

func ParseImports(paths ...string) ([]*build.Package, error) {
	paths, err := ExpandPaths(paths...)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	// Parse the files in each of the directories and mark the ones we
	// haven't listed as unvisited.
	visited := make(map[dependency]struct{})
	unvisited := make(map[dependency]build.ImportMode)
	for _, path := range paths {
		// Resolve to the absolute path so this matches later implicit imports.
		abspath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		// Import the current package from the absolute path. Ignore vendored
		// dependencies just in case we have vendored ourselves.
		pkg, err := build.Import(".", abspath, build.IgnoreVendor)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				continue
			}
			return nil, err
		}

		ignored := make(map[string]struct{}, len(pkg.IgnoredGoFiles))
		for _, file := range pkg.IgnoredGoFiles {
			ignored[file] = struct{}{}
		}
		visited[dependency{pkg.ImportPath, abspath}] = struct{}{}

		pkgs, err := parser.ParseDir(fset, abspath, func(fi os.FileInfo) bool {
			_, ok := ignored[fi.Name()]
			return !ok
		}, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}

		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, imp := range file.Imports {
					name := strings.Trim(imp.Path.Value, `"`)
					unvisited[dependency{name, abspath}] = build.IgnoreVendor
				}
			}
		}
	}

	// Parse the unvisited paths to resolve the dependency to a path.
	var packages []*build.Package
	for len(unvisited) > 0 {
		for dep, opt := range unvisited {
			delete(unvisited, dep)

			// Resolve the dependency to an import path.
			pkg, err := build.Import(dep.Name, dep.Dir, opt)
			if err != nil {
				if _, ok := err.(*build.NoGoError); ok {
					continue
				}
				return nil, err
			}

			// If we have seen the resolved dependency before,
			// short circuit here.
			resolvedDep := dependency{pkg.ImportPath, pkg.Dir}
			if _, ok := visited[resolvedDep]; ok {
				continue
			}

			// We have never seen this dependency. Add it to the list of
			// dependencies we have seen.
			// We may see more than one package with the same import path.
			packages = append(packages, pkg)
			visited[resolvedDep] = struct{}{}

			// Make a set of the go files that were ignored so we don't
			// parse those files.
			ignored := make(map[string]struct{}, len(pkg.IgnoredGoFiles))
			for _, file := range pkg.IgnoredGoFiles {
				ignored[file] = struct{}{}
			}
			// Ignore test files from dependent packages.
			for _, file := range pkg.TestGoFiles {
				ignored[file] = struct{}{}
			}
			for _, file := range pkg.XTestGoFiles {
				ignored[file] = struct{}{}
			}

			// Parse the imports for this package and add to unvisited.
			pkgs, err := parser.ParseDir(fset, pkg.Dir, func(fi os.FileInfo) bool {
				_, ok := ignored[fi.Name()]
				return !ok
			}, parser.ImportsOnly)
			if err != nil {
				return nil, err
			}

			newpkgs := make(map[dependency]struct{})
			for _, p := range pkgs {
				for _, file := range p.Files {
					for _, imp := range file.Imports {
						path, err := strconv.Unquote(imp.Path.Value)
						if err != nil || path == "C" {
							continue
						}

						dep := dependency{path, pkg.Dir}
						if _, ok := newpkgs[dep]; !ok {
							unvisited[dep] = 0
							newpkgs[dep] = struct{}{}
						}
					}
				}
			}
		}
	}
	return packages, nil
}
