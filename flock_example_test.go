// Copyright 2015 Tim Heckman. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package flock_test

import (
	"fmt"

	"github.com/theckman/go-flock"
)

func ExampleFlock_TryLock() {
	// should probably put these in /var/lock
	fileLock := flock.NewFlock("/tmp/go-lock.lock")

	locked, err := fileLock.TryLock()

	if err != nil {
		// handle locking error
	}

	if locked {
		fmt.Printf("path: %s; locked: %v\n", fileLock.Path(), fileLock.Locked())

		if err := fileLock.Unlock(); err != nil {
			// handle unlock error
		}
	}

	fmt.Printf("path: %s; locked: %v\n", fileLock.Path(), fileLock.Locked())
	// Output: path: /tmp/go-lock.lock; locked: true
	// path: /tmp/go-lock.lock; locked: false
}
