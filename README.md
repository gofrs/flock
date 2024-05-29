# flock

[![Go Reference](https://pkg.go.dev/badge/github.com/gofrs/flock.svg)](https://pkg.go.dev/github.com/gofrs/flock)
[![License](https://img.shields.io/badge/license-BSD_3--Clause-brightgreen.svg?style=flat)](https://github.com/gofrs/flock/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/gofrs/flock)](https://goreportcard.com/report/github.com/gofrs/flock)

`flock` implements a thread-safe sync.Locker interface for file locking.
It also includes a non-blocking `TryLock()` function to allow locking without blocking execution.

## License

`flock` is released under the BSD 3-Clause License. See the `LICENSE` file for more details.

## Go Compatibility

This package makes use of the `context` package that was introduced in Go 1.7.
As such, this package has an implicit dependency on Go 1.7+.

## Installation

```bash
go get -u github.com/gofrs/flock
```

## Usage

```go
import "github.com/gofrs/flock"

fileLock := flock.New("/var/lock/go-lock.lock")

locked, err := fileLock.TryLock()

if err != nil {
	// handle locking error
}

if locked {
	// do work
	fileLock.Unlock()
}
```

For more detailed usage information take a look at the package API docs on
[GoDoc](https://pkg.go.dev/github.com/gofrs/flock).
