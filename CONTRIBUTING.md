Contributing
============

# Developing locally

> You'll need at least version 1.21 of Golang.

To build the CLI locally:

```shell
go build
```

To run the tests locally:

```shell
go test -v ./...
```

The installer is shell, not Go, and has its own suite. It fakes `uname` and
serves a release fixture over `file://`, so it needs no network:

```shell
./install_test.sh
```

## Adding a command

# Releasing

See [RELEASING.md](RELEASING.md).
