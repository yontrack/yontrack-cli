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

Two of the scripts are shell rather than Go, and have their own suites. Both
report `ok`/`FAIL` per case and exit non-zero if anything failed.

```shell
./install_test.sh   # the installer: fakes uname, serves a fixture over file://
./build_test.sh     # the release build: runs it, then checks what it produced
```

`install_test.sh` needs no network. `build_test.sh` runs the real build, so it
takes about half a minute and replaces whatever artifacts are in the working
directory. Shared helpers live in `test_lib.sh`.

## Adding a command

# Releasing

See [RELEASING.md](RELEASING.md).
