goveralls
=========

[Go](http://golang.org) integration for [Coveralls.io](http://coveralls.io)
continuous code coverage tracking system.

# Installation

`goveralls` requires a working Go installation (Go1 or higher).

```bash
$ go get github.com/mattn/goveralls
```


# Usage

First you will need an API token.  It is found at the bottom of your
repository's page when you are logged in to Coveralls.io.  Each repo has its
own token.

```bash
$ cd $GOPATH/src/github.com/yourusername/yourpackage
$ goveralls your_repos_coveralls_token
```

# Continuous Integration

There is no need to run `go test` separately, as `goveralls` runs the entire
test suite.

## Travis CI

`goveralls` currently cannot be used with Travis.  This may change when Go 1.1 is released.


## Drone.io

Store your Coveralls API token in `Enviornment Variables`:

```
COVERALLS_TOKEN=your_token_goes_here
```

Replace the `go test` line in your `Commands` with these lines:

```
go get github.com/axw/gocov/gocov
go get github.com/mattn/goveralls
goveralls -service drone.io $COVERALLS_TOKEN
```

You can use the `-v` flag to see verbose output from the test suite:

```
goveralls -v -service drone.io $COVERALLS_TOKEN
```


# License

under the MIT License: http://mattn.mit-license.org/2013

