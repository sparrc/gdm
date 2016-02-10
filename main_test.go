package main

import (
	"os"
	"testing"
)

func TestGetGoPath(t *testing.T) {
	tmpdir := os.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %s", err.Error())
	}
	origGopath := os.Getenv("GOPATH")

	// Verify that getGoPath works
	_, err = getGoPath(wd)
	if err != nil {
		t.Errorf("Expected getGoPath success, got %s", err.Error())
	}

	// Unset GOPATH and verify that getGoPath fails
	err = os.Unsetenv("GOPATH")
	if err != nil {
		t.Fatal(err.Error())
	}
	_, err = getGoPath(wd)
	if err == nil {
		t.Errorf("Expected getGoPath failure, got %s", err)
	}

	// Set gopath to tmp directory and verify that getGoPath fails
	err = os.Setenv("GOPATH", tmpdir)
	if err != nil {
		t.Fatal(err.Error())
	}
	_, err = getGoPath(wd)
	if err == nil {
		t.Errorf("Expected getGoPath failure, got %s", err)
	}

	// Set gopath to GOPATH + tmp directory and verify that getGoPath succeeds
	err = os.Setenv("GOPATH", origGopath+":"+tmpdir)
	if err != nil {
		t.Fatal(err.Error())
	}
	_, err = getGoPath(wd)
	if err != nil {
		t.Errorf("Expected getGoPath failure, got %s", err.Error())
	}
}
