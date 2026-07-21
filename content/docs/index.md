---
title: Install
slug: install
group: Start here
weight: 1
description: Install the Fabrik CLI with the one-line installer, or from source with go install.
lede: "The one-line installer is the quickest path: it downloads the latest prebuilt CLI for your platform, verifies its checksum, and puts it on your PATH."
---

## Prerequisites {#prerequisites}

Fabrik builds and runs ordinary Go applications, so it needs the Go toolchain:
**Go 1.26 or newer**. If you don't have it yet, install it from the
[Go install page](https://go.dev/doc/install).

## Install the CLI {#install-cli}

```
curl -fsSL https://gofabrik.dev/install.sh | sh
```

## Install from source {#install-source}

If you'd rather build from source with the Go toolchain, use `go install`
and make sure the Go bin directory is on your `PATH`.

Install the CLI with Go:

```
go install github.com/gofabrik/fabrik/fabrik@latest
```

`go install` places `fabrik` in Go's bin directory (`$(go env GOPATH)/bin`,
e.g. `~/go/bin`). Add it for your shell:

```
# zsh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc && source ~/.zshrc

# bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc && source ~/.bashrc

# fish
fish_add_path (go env GOPATH)/bin
```

Verify:

```
fabrik
# fabrik new hello
```
