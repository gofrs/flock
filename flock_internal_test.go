// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2026 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

//go:build !js && !plan9 && !wasip1

package flock

import (
	"os"
	"runtime"
	"testing"
)

func TestFlock_fh_onError(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "go-flock-")
	requireNoError(t, err)

	err = tmpFile.Close()
	requireNoError(t, err)

	err = os.Remove(tmpFile.Name())
	requireNoError(t, err)

	lock := New(tmpFile.Name())

	locked, err := lock.TryLock()
	requireNoError(t, err)

	if !locked {
		t.Fatal("TryLock() = false, want true")
	}

	newLock := New(tmpFile.Name())

	locked, err = newLock.TryLock()
	requireNoError(t, err)

	if locked {
		t.Error("contending TryLock() = true, want false")
	}

	if newLock.fh != nil {
		t.Error("file handle should have been released and be nil")
	}

	err = lock.Unlock()
	requireNoError(t, err)
}

func TestFlock_fh_onError_dir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on Windows")
	}

	tmpDir := t.TempDir()

	lock := New(tmpDir, SetFlag(os.O_RDONLY))

	locked, err := lock.TryLock()
	requireNoError(t, err)

	if !locked {
		t.Fatal("TryLock() = false, want true")
	}

	newLock := New(tmpDir, SetFlag(os.O_RDONLY))

	locked, err = newLock.TryLock()
	requireNoError(t, err)

	if locked {
		t.Error("contending TryLock() = true, want false")
	}

	if newLock.fh != nil {
		t.Error("file handle should have been released and be nil")
	}

	err = lock.Unlock()
	requireNoError(t, err)
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
