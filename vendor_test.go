package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestExpandPaths(t *testing.T) {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "gdm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	os.MkdirAll(filepath.Join(tmpdir, "a/b"), os.ModePerm)
	os.MkdirAll(filepath.Join(tmpdir, "vendor"), os.ModePerm)

	paths, err := ExpandPaths("/usr/local", filepath.Join(tmpdir, "..."))
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"/usr/local",
		tmpdir,
		filepath.Join(tmpdir, "a"),
		filepath.Join(tmpdir, "a/b"),
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("unexpected paths: %#v != %#v", paths, want)
	}
}

func TestParseImports(t *testing.T) {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "gdm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	const repoPath = "github.com/sparrc/imports"
	for _, f := range []struct {
		path    string
		content string
	}{
		{
			path: "a.go",
			content: `package main
import (
	"fmt"
	"golang.org/x/tools/cover"
	"golang.org/x/tools/refactor" // empty directory
)`,
		},
		{
			path: "a/b.go",
			content: `package a
import "golang.org/x/tools/go/vcs"
`,
		},
	} {
		filename := filepath.Join(tmpdir, repoPath, f.path)
		if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
			t.Fatalf("unable to make directory %s: %s", filepath.Dir(filename), err)
		}

		fd, err := os.Create(filename)
		if err != nil {
			t.Fatalf("unable to create file %s: %s", filename, err)
		}

		fd.WriteString(f.content)
		fd.Close()
	}

	pkgs, err := ParseImports(filepath.Join(tmpdir, repoPath, "..."))
	if err != nil {
		t.Fatalf("unable to parse imports: %s", err)
	}

	got := make([]string, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.Goroot {
			continue
		}
		got = append(got, pkg.ImportPath)
	}
	sort.Strings(got)

	want := []string{
		"golang.org/x/tools/cover",
		"golang.org/x/tools/go/vcs",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected paths: %#v != %#v", got, want)
	}
}
