package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetRootImport(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	s := "github.com/sparrc/gdm"
	rootImport := getRootImport(wd)
	if rootImport != s {
		t.Errorf("Expected rootImport %s, got %s", s, rootImport)
	}
}

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
	if len(imports) != 20 {
		t.Errorf("Expected %d imports, got %d", 20, len(imports))
	}

	tests := []struct {
		importpath string
		rev        string
	}{
		{"collectd.org/api", "9fc824c70f713ea0f058a07b49a4c563ef2a3b98"},
		{"collectd.org/network", "9fc824c70f713ea0f058a07b49a4c563ef2a3b98"},
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
