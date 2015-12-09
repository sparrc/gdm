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

### Building

One of the goals of gdm is to do as little as possible, letting the standard go
build tools take care of most things. Building a gdm project looks like this:

1. Get project: `go get github.com/foo/bar/...`
1. Go to project dir: `cd $GOPATH/src/github.com/foo/bar`
1. Restore dependency versions: `gdm restore`
1. Build `go build` or `go build ./...`

### Add a Dependency

To add a new package github.com/foo/bar, do this:

1. Run `go get github.com/foo/bar`
1. Edit your code to import github.com/foo/bar.
1. Run `gdm save`

### Update a Dependency

To update a package from your `$GOPATH`, do this:

1. Run `go get -u github.com/foo/bar`
1. Run `gdm save`.

Before committing the change, you'll probably want to inspect the changes to
Godeps, for example with `git diff`, and make sure it looks reasonable.

### Update all dependencies

To update all dependencies from your `$GOPATH`, do this:

1. Run `go get -u ./...`
1. Run `gdm save`.

### Migrating from other dependency management solutions

- [gpm](https://github.com/pote/gpm)
    - Nothing! the gdm `Godeps` file is the same as gpm's
- [godep](https://github.com/tools/godep)
    - NOTE: only do this if you haven't used the godep import rewriting feature
    - Run `rm -rf Godeps/`
    - Run `gdm save`

### Acknowledgements

If you're familiar with Go dependency management, you can probably see the
similarities with [gpm](https://github.com/pote/gpm) and
[godep](https://github.com/tools/godep). This tool could not have existed without
their influence! The main differences/similarities are that `gdm` aims to be
minimalist (like gpm) and written in Go (like godep)
