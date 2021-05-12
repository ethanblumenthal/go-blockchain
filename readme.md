# GoChain

## Installation

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

## Usage

### Install

```
go install ./cmd/...
```

## CLI

### Show available commands and flags

```bash
gochain help
```

#### Show available run settings

```bash
gochain run --help

Launches the GoChain node and its HTTP API.

Usage:
  gochain run [flags]

Flags:
      --bootstrap-account string   default GoChain bootstrap's Genesis account with 1M GoChain tokens (default "0x09ee50f2f37fcba1845de6fe5c762e83e65e755c")
      --bootstrap-ip string        default GoChain bootstrap's server to interconnect peers (default "node.gochain.bootstrap")
      --bootstrap-port uint        default GoChain bootstrap's server port to interconnect peers (default 443)
      --datadir string             Absolute path to your node's data dir where the DB will be/is stored
      --disable-ssl                should the HTTP API SSL certificate be disabled? (default false)
  -h, --help                       help for run
      --ip string                  your node's public IP to communication with other peers (default "127.0.0.1")
      --miner string               your node's miner account to receive the block rewards (default "0x0000000000000000000000000000000000000000")
      --port uint                  your node's public HTTP port for communication with other peers (configurable if SSL is disabled) (default 443)
```

### Run a GoChain node connected to the official GoChain test network

If you are running the node on your localhost, just disable the SSL with `--disable-ssl` flag.

```
gochain version
> Version: 1.0.0-beta GoChain Ledger

gochain run --datadir=$HOME/.gochain --ip=127.0.0.1 --port=8081 --miner=0x_YOUR_WALLET_ACCOUNT --disable-ssl
```

### Run a GoChain bootstrap node in isolation, on your localhost only

```
gochain run --datadir=$HOME/.gochain_boostrap --ip=127.0.0.1 --port=8080 --bootstrap-ip=127.0.0.1 --bootstrap-port=8080 --disable-ssl
```

#### Run a second GoChain node connecting to your first one

```
gochain run --datadir=$HOME/.gochain --ip=127.0.0.1 --port=8081 --bootstrap-ip=127.0.0.1 --bootstrap-port=8080 --disable-ssl
```

### Create a new account

```
gochain wallet new-account --datadir=$HOME/.gochain
```

### Run a GoChain node with SSL

The default node's HTTP port is 443. The SSL certificate is generated automatically as long as the DNS A/AAAA records point at your server.

#### Official Testing Bootstrap Server

Example how the official GoChain bootstrap node is launched. Customize the `--datadir`, `--miner`, and `--ip` values to match your server.

```bash
/usr/local/bin/gochain run --datadir=/home/ec2-user/.gochain --miner=0x09ee50f2f37fcba1845de6fe5c762e83e65e755c --ip=node.gochain.bootstrap --port=443 --ssl-email=ethan.blumenthal@gmail.com --bootstrap-ip=node.gochain.bootstrap --bootstrap-port=443 --bootstrap-account=0x09ee50f2f37fcba1845de6fe5c762e83e65e755c
```

## HTTP

### List all balances

```
curl http://localhost:8080/balances/list | jq
```

### Send a signed TX

```
curl --location --request POST 'http://localhost:8080/tx/add' \
--header 'Content-Type: application/json' \
--data-raw '{
	"from": "0x22ba1f80452e6220c7cc6ea2d1e3eeddac5f694a",
	"from_pwd": "security123",
	"to": "0x6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8",
	"value": 100
}'
```

### Check node's status (latest block, known peers, pending TXs)

```
curl http://localhost:8080/node/status | jq
```

## Tests

Run all tests with verbosity but one at a time, without timeout, to avoid ports collisions:

```
go test -v -p=1 -timeout=0 ./...
```

Run an individual test:

```
go test -timeout=0 ./node -test.v -test.run ^TestNode_Mining$
```

**Note:** Majority are integration tests and take time. Expect the test suite to finish in ~30 mins.
