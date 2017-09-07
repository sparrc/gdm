# Go Dependency Manager (gdm) [![Circle CI](https://circleci.com/gh/sparrc/gdm.svg?style=svg)](https://circleci.com/gh/sparrc/gdm)

gdm aims to do as little as possible. It will checkout dependencies to the
local vendor directory and does not require that people use `gdm` to build
your project. In this way, people can still simply `go get` your project
and build.

We would recommend that you add `vendor` to your .gitignore file when using gdm.

This tool assumes you are working in a standard Go workspace, as described in
http://golang.org/doc/code.html.

### Install

```
go get github.com/sparrc/gdm
```

### How to use gdm with a new project

Assuming your Go workspace is setup, so you can build your project
with `go install` or `go install ./...`, it's one command to start using:

```
gdm save
```

This will create a new file in your repo directory called `Godeps`, which
specifies project dependencies and their revisions. This file is identical to
the file used by [gpm](https://github.com/pote/gpm).

Godeps is a simple text file of repo roots and revisions:

```
collectd.org/api 9fc824c70f713ea0f058a07b49a4c563ef2a3b98
collectd.org/network 9fc824c70f713ea0f058a07b49a4c563ef2a3b98
github.com/BurntSushi/toml 056c9bc7be7190eaa7715723883caffa5f8fa3e4
```

The file supports comments using the `#` character.

## Vendor

The `gdm vendor` command is the opposite of `gdm save`. It will checkout the
package versions specified in Godeps to the vendor directory.

### Add a Dependency

To add a new package github.com/foo/bar, do this:

1. Run `go get github.com/foo/bar`
1. Run `gdm save`

### Update a Dependency

To update a package to the latest version, do this:

1. Run `rm -rf ./vendor`
1. Run `go get -u github.com/foo/bar`
1. Run `gdm save`

Before committing the change, you'll probably want to inspect the changes to
Godeps, for example with `git diff`, and make sure it looks reasonable.

### Update all dependencies

To update all dependencies from your `$GOPATH`, do this:

1. Run `rm -rf ./vendor`
1. Run `go get -u ./...`
1. Run `gdm save`

### Building a gdm project

Building a project managed by gdm looks like this:

1. Run `go get github.com/foo/bar`
1. Run `cd $GOPATH/src/github.com/foo/bar`
1. Run `gdm vendor`
1. Build: `go install ./...`

## Homebrew

To help make a [homebrew](https://github.com/Homebrew/homebrew)
formula for your Go project, gdm supports a `gdm brew` command, which will print
out your dependencies to stdout in the homebrew go_resource format, like this:

```console
$ gdm brew
  go_resource "collectd.org/api" do
    url "https://github.com/collectd/go-collectd.git",
    :revision => "9fc824c70f713ea0f058a07b49a4c563ef2a3b98"
  end

  go_resource "collectd.org/network" do
    url "https://github.com/collectd/go-collectd.git",
    :revision => "9fc824c70f713ea0f058a07b49a4c563ef2a3b98"
  end

  go_resource "github.com/BurntSushi/toml" do
    url "https://github.com/BurntSushi/toml.git",
    :revision => "056c9bc7be7190eaa7715723883caffa5f8fa3e4"
  end

  ...
```

### Restore

The `gdm restore` command works similar to the `gdm vendor` command, but instead
of checking out the dependencies in the ./vendor directory, it will checkout the
dependencies in your current GOPATH. This will modify repos in your GOPATH.

This can be useful for debugging or if you are using a Go version earlier than
1.9.

#### Acknowledgements

If you're familiar with Go dependency management, you can probably see the
similarities with [gpm](https://github.com/pote/gpm) and
[godep](https://github.com/tools/godep). This tool could not have existed without
their influence!
