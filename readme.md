# Installation

### Install Go 1.16 or higher

Follow the official docs or use your favorite dependency manager
to install Go: [https://golang.org/doc/install](https://golang.org/doc/install)

Verify your `$GOPATH` is correctly set before continuing!

### Setup this repository

Go is bit picky about where you store your repositories.

The convention is to store:

- the source code inside the `$GOPATH/src`
- the compiled program binaries inside the `$GOPATH/bin`

You can `clone` the repository or use `go get` to install it.

#### Using Git

```bash
mkdir -p $GOPATH/src/github.com/ethanblumenthal
cd $GOPATH/src/github.com/ethanblumenthal

git clone git@github.com:ethanblumenthal/golang-blockchain.git
```

PS: Make sure you actually clone it inside the `src/github.com/ethanblumenthal` directory, not your own, otherwise it won't compile. Go rules.

#### Using Go get

```bash
go get -u github.com/ethanblumenthal/golang-blockchain
```

# Usage

## Install

```
go install ./cmd/...
```

## CLI

### Show available commands and flags

```bash
blockchain help
```
