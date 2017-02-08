package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/vcs"
)

func TestGetRepoRoot(t *testing.T) {
	s := "github.com/sparrc/gdm"
	_, err := getRepoRoot(s)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestImportsFromFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	filename := filepath.Join(wd, "test", "TestGodeps")
	imports := ImportsFromFile(filename)
	if len(imports) != 19 {
		t.Errorf("Expected %d imports, got %d", 20, len(imports))
	}

	tests := []struct {
		importpath string
		rev        string
	}{
		{"collectd.org/api", "9fc824c70f713ea0f058a07b49a4c563ef2a3b98"},
		// This import is in file but has the same "repo root" as collectd.org/api
		// so it shouldn't show up in the 'restore' import paths
		// {"collectd.org/network", "9fc824c70f713ea0f058a07b49a4c563ef2a3b98"},
		{"github.com/BurntSushi/toml", "056c9bc7be7190eaa7715723883caffa5f8fa3e4"},
		{"github.com/bmizerany/pat", "b8a35001b773c267eb260a691f4e5499a3531600"},
		{"github.com/boltdb/bolt", "b34b35ea8d06bb9ae69d9a349119252e4c1d8ee0"},
		{"github.com/davecgh/go-spew", "5215b55f46b2b919f50a1df0eaa5886afe4e3b3d"},
		{"github.com/dgryski/go-bits", "86c69b3c986f9d40065df5bd8f765796549eef2e"},
		{"github.com/dgryski/go-bitstream", "27cd5973303fde7d914860be1ea4b927a6be0c92"},
		{"github.com/gogo/protobuf", "e492fd34b12d0230755c45aa5fb1e1eea6a84aa9"},
		{"github.com/golang/snappy", "723cc1e459b8eea2dea4583200fd60757d40097a"},
		{"github.com/hashicorp/raft", "d136cd15dfb7876fd7c89cad1995bc4f19ceb294"},
		{"github.com/hashicorp/raft-boltdb", "d1e82c1ec3f15ee991f7cc7ffd5b67ff6f5bbaee"},
		{"github.com/influxdb/enterprise-client", "25665cba4f54fa822546c611c9414ac31aa10faa"},
		{"github.com/jwilder/encoding", "07d88d4f35eec497617bee0c7bfe651a796dae13"},
		{"github.com/kimor79/gollectd", "61d0deeb4ffcc167b2a1baa8efd72365692811bc"},
		{"github.com/paulbellamy/ratecounter", "5a11f585a31379765c190c033b6ad39956584447"},
		{"github.com/peterh/liner", "4d47685ab2fd2dbb46c66b831344d558bc4be5b9"},
		{"github.com/rakyll/statik", "274df120e9065bdd08eb1120e0375e3dc1ae8465"},
		{"golang.org/x/crypto", "7b85b097bf7527677d54d3220065e966a0e3b613"},
		{"gopkg.in/fatih/pool.v2", "cba550ebf9bce999a02e963296d4bc7a486cb715"},
	}

	for i, test := range tests {
		i := imports[i]
		if i.ImportPath != test.importpath {
			t.Errorf("Expected %s, actual %s", test.importpath, i.ImportPath)
		}
		if i.Rev != test.rev {
			t.Errorf("Expected %s, actual %s", test.rev, i.Rev)
		}
	}
}

func TestImport_RestoreImport(t *testing.T) {
	// Restore gdm import to specific known revision.
	I := Import{
		ImportPath: "github.com/sparrc/gdm",
		Rev:        "2b7833e0c093654450c829858d0df5c53c575b07",
		Repo: &vcs.RepoRoot{
			VCS:  vcs.ByCmd("git"),
			Repo: "git://github.com/sparrc/gdm",
			Root: "github.com/sparrc/gdm",
		},
	}

	tmpdir, err := ioutil.TempDir(os.TempDir(), "gdm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	srcdir := filepath.Join(tmpdir, "src", I.ImportPath)
	if err := I.RestoreImport(tmpdir); err != nil {
		t.Fatal(err)
	}
	if rev := getRevisionFromPath(srcdir, I.Repo); rev != I.Rev {
		t.Fatalf("unable to restore import to revision %s", I.Rev)
	}

	// Restore to the previous revision to test restoring an import on disk.
	cmd := exec.Command("git", "rev-parse", "HEAD~1")
	cmd.Dir = srcdir

	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	I.Rev = strings.TrimSpace(string(output))
	if err := I.RestoreImport(tmpdir); err != nil {
		t.Fatal(err)
	}
	if rev := getRevisionFromPath(srcdir, I.Repo); rev != I.Rev {
		t.Fatalf("unable to restore import to revision %s", I.Rev)
	}
}
