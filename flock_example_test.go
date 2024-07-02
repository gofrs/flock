// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2024 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

//go:build !js && !plan9 && !wasip1

package flock_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

func ExampleFlock_Locked() {
	f := flock.New(filepath.Join(os.TempDir(), "go-lock.lock"))

	_, err := f.TryLock()
	if err != nil {
		// handle locking error
		panic(err)
	}

	fmt.Printf("locked: %v\n", f.Locked())

	err = f.Unlock()
	if err != nil {
		// handle locking error
		panic(err)
	}

	fmt.Printf("locked: %v\n", f.Locked())

	// Output: locked: true
	// locked: false
}

func ExampleFlock_TryLock() {
	// should probably put these in /var/lock
	f := flock.New(filepath.Join(os.TempDir(), "go-lock.lock"))

	locked, err := f.TryLock()
	if err != nil {
		// handle locking error
		panic(err)
	}

	if locked {
		fmt.Printf("path: %s; locked: %v\n", f.Path(), f.Locked())

		if err := f.Unlock(); err != nil {
			// handle unlock error
			panic(err)
		}
	}

	fmt.Printf("path: %s; locked: %v\n", f.Path(), f.Locked())
}

func ExampleFlock_TryLockContext() {
	// should probably put these in /var/lock
	f := flock.New(filepath.Join(os.TempDir(), "go-lock.lock"))

	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	locked, err := f.TryLockContext(lockCtx, 678*time.Millisecond)
	if err != nil {
		// handle locking error
		panic(err)
	}

	if locked {
		fmt.Printf("path: %s; locked: %v\n", f.Path(), f.Locked())

		if err := f.Unlock(); err != nil {
			// handle unlock error
			panic(err)
		}
	}

	fmt.Printf("path: %s; locked: %v\n", f.Path(), f.Locked())
}
