package flock

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test(t *testing.T) {
	tmpFileFh, err := ioutil.TempFile(os.TempDir(), "go-flock-")
	tmpFileFh.Close()
	tmpFile := tmpFileFh.Name()
	os.Remove(tmpFile)

	lock := New(tmpFile)
	locked, err := lock.TryLock()
	if locked == false || err != nil {
		t.Fatal("failed to lock")
	}

	newLock := New(tmpFile)
	locked, err = newLock.TryLock()
	if locked != false || err != nil {
		t.Fatal("should have failed locking")
	}

	if newLock.fh != nil {
		t.Fatal("file handle should have been released and be nil")
	}
}
