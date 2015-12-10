## Go Dependency Manager (gdm) [![Circle CI](https://circleci.com/gh/sparrc/gdm.svg?style=svg)](https://circleci.com/gh/sparrc/gdm)

gdm is a lightweight package manager for Go written in Go. It does not copy
dependencies in-repo and does not require that people (users and developers)
use `gdm` to build your project. In this way, people can still simply `go get`
your project and build it.

This tool assumes you are working in a standard Go workspace, as described in
http://golang.org/doc/code.html.

### Install

```console
$ go get github.com/sparrc/gdm
```

### How to use gdm with a new project

Assuming you've got everything working already, so you can build your project
with `go install` or `go install ./...`, it's one command to start using:

```console
$ gdm save
```

This will create a new file in your repo directory called `Godeps`, which
specifies project dependencies and their revisions. This file is identical to
the file used by [gpm](https://github.com/pote/gpm).

`Godeps` is a simple text file of repo roots and revisions,
also supporting comments:

```
  # collectd things:
  collectd.org/api 9fc824c70f713ea0f058a07b49a4c563ef2a3b98
  collectd.org/network 9fc824c70f713ea0f058a07b49a4c563ef2a3b98

  # toml parser:
  github.com/BurntSushi/toml 056c9bc7be7190eaa7715723883caffa5f8fa3e4
  ...
```

### Restore

The `gdm restore` command is the opposite of `gdm save`. It will install the
package versions specified in `Godeps` to your `$GOPATH`. This modifies the
state of packages in your `$GOPATH`.

### Add a Dependency

To add a new package github.com/foo/bar, do this:

1. Run `gdm restore`
1. Run `go get github.com/foo/bar`
1. Run `gdm save`

### Update a Dependency

To update a package from your `$GOPATH`, do this:

1. Run `gdm restore`
1. Run `go get -u github.com/foo/bar`
1. Run `gdm save`

Before committing the change, you'll probably want to inspect the changes to
Godeps, for example with `git diff`, and make sure it looks reasonable.

### Update all dependencies

To update all dependencies from your `$GOPATH`, do this:

1. Run `go get -u ./...`
1. Run `gdm save`

### Building a gdm project

Building a project managed by gdm looks like this:

1. Run `go get github.com/foo/bar`
1. Run `cd $GOPATH/src/github.com/foo/bar`
1. Run `gdm restore`
1. Build: `go install` or `go install ./...`

### Homebrew

To help making a homebrew formula for your go project, gdm supports a
`gdm brew` command, which will print out your dependencies to stdout in the
homebrew go_resource format, like this:

```
$ gdm brew
======= Go Dependency Manager =======
= working dir: /Users/csparr/ws/go/src/github.com/sparrc/gdm
= GOPATH:      /Users/csparr/ws/go
=====================================

  go_resource "golang.org/x/tools" do
    url "https://go.googlesource.com/tools.git",
    :revision => "b48dc8da98ae78c3d11f220e7d327304c84e623a"
  end

  ...
```

#### Acknowledgements

If you're familiar with Go dependency management, you can probably see the
similarities with [gpm](https://github.com/pote/gpm) and
[godep](https://github.com/tools/godep). This tool could not have existed without
their influence!
