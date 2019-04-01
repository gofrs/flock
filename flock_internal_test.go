package flock

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"
)

func Test(t *testing.T) {
	pathsToLock := make([]string, 1, 2)

	tmpFileFh, _ := ioutil.TempFile(os.TempDir(), "go-flock-")
	tmpFileFh.Close()
	tmpFile := tmpFileFh.Name()
	os.Remove(tmpFile)
	pathsToLock[0] = tmpFile

	if runtime.GOOS == "linux" {
		tmpDir, _ := ioutil.TempDir("", "go-flock-")
		os.Remove(tmpDir)
		pathsToLock = append(pathsToLock, tmpDir)
	}

	for _, path := range pathsToLock {
		lock := New(path)
		locked, err := lock.TryLock()
		if locked == false || err != nil {
			t.Fatalf("failed to lock: locked: %t, err: %v", locked, err)
		}

		newLock := New(path)
		locked, err = newLock.TryLock()
		if locked != false || err != nil {
			t.Fatalf("should have failed locking: locked: %t, err: %v", locked, err)
		}

		if newLock.fh != nil {
			t.Fatal("file handle should have been released and be nil")
		}
	}
}
