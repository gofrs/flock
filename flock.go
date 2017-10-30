// Copyright 2015 Tim Heckman. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

// Package flock implements a thread-safe sync.Locker interface for file locking.
// It also includes a non-blocking TryLock() function to allow locking
// without blocking execution.
//
// Package flock is released under the BSD 3-Clause License. See the LICENSE file
// for more details.
package flock

import (
	"context"
	"os"
	"sync"
	"time"
)

// Flock is the struct type to handle file locking. All fields are unexported,
// with access to some of the fields provided by getter methods (Path() and Locked()).
type Flock struct {
	path string
	m    sync.RWMutex
	fh   *os.File
	l    bool
}

// NewFlock is a function to return a new instance of *Flock. The only parameter
// it takes is the path to the desired lockfile.
func NewFlock(path string) *Flock {
	return &Flock{path: path}
}

// Path is a function to return the path as provided in NewFlock().
func (f *Flock) Path() string {
	return f.path
}

// Locked is a function to return the current lock state (locked: true, unlocked: false).
func (f *Flock) Locked() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.l
}

func (f *Flock) String() string {
	return f.path
}

// TryLockContext repeatedly tries locking until one of the conditions is met:
// TryLock succeeds, TryLock fails with error, or Context Done channel is closed.
func (f *Flock) TryLockContext(ctx context.Context, retryDelay time.Duration) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	for {
		if ok, err := f.TryLock(); ok || err != nil {
			return ok, err
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(retryDelay):
			// try again
		}
	}
}

func (f *Flock) setFh() error {
	// open a new os.File instance
	// create it if it doesn't exist, truncate it if it does exist, open the file read-write
	fh, err := os.OpenFile(f.path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(0600))

	if err != nil {
		return err
	}

	// set the filehandle on the struct
	f.fh = fh
	return nil
}
