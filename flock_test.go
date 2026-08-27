// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2026 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

//go:build !js && !plan9 && !wasip1

package flock_test

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

type TestSuite struct {
	t *testing.T

	dir  bool
	opts []flock.Option

	path  string
	flock *flock.Flock
}

func Test(t *testing.T) {
	runTests(t, false, nil)
}

func Test_dir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on Windows")
	}

	runTests(t, true, []flock.Option{flock.SetFlag(os.O_RDONLY)})
}

func (s *TestSuite) SetupTest() {
	if s.dir {
		s.path = s.t.TempDir()

		s.flock = flock.New(s.path, s.opts...)

		return
	}

	tmpFile, err := os.CreateTemp(s.t.TempDir(), "go-flock-")
	s.requireNoError(err)

	s.path = tmpFile.Name()

	err = tmpFile.Close()
	s.requireNoError(err)

	err = os.Remove(s.path)
	s.requireNoError(err)

	s.flock = flock.New(s.path, s.opts...)
}

func (s *TestSuite) TearDownTest() {
	_ = s.flock.Unlock()
	_ = os.Remove(s.path)
}

func runTests(t *testing.T, dir bool, opts []flock.Option) {
	t.Helper()

	tests := []struct {
		name string
		run  func(*TestSuite)
	}{
		{"TestNew", (*TestSuite).TestNew},
		{"TestFlock_Path", (*TestSuite).TestFlock_Path},
		{"TestFlock_Locked", (*TestSuite).TestFlock_Locked},
		{"TestFlock_RLocked", (*TestSuite).TestFlock_RLocked},
		{"TestFlock_String", (*TestSuite).TestFlock_String},
		{"TestFlock_TryLock", (*TestSuite).TestFlock_TryLock},
		{"TestFlock_TryRLock", (*TestSuite).TestFlock_TryRLock},
		{"TestFlock_TryLockContext", (*TestSuite).TestFlock_TryLockContext},
		{"TestFlock_TryRLockContext", (*TestSuite).TestFlock_TryRLockContext},
		{"TestFlock_Unlock", (*TestSuite).TestFlock_Unlock},
		{"TestFlock_Lock", (*TestSuite).TestFlock_Lock},
		{"TestFlock_RLock", (*TestSuite).TestFlock_RLock},
		{"TestFlock_Stat", (*TestSuite).TestFlock_Stat},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &TestSuite{t: t, dir: dir, opts: opts}
			s.SetupTest()
			t.Cleanup(s.TearDownTest)
			tc.run(s)
		})
	}
}

func (s *TestSuite) TestNew() {
	f := flock.New(s.path, s.opts...)
	if f == nil {
		s.t.Fatal("New() returned nil")
	}

	if got := f.Path(); got != s.path {
		s.t.Errorf("Path() = %q, want %q", got, s.path)
	}

	s.checkLockState(f, false, false)
}

func (s *TestSuite) TestFlock_Path() {
	if got := s.flock.Path(); got != s.path {
		s.t.Errorf("Path() = %q, want %q", got, s.path)
	}
}

func (s *TestSuite) TestFlock_Locked() {
	if s.flock.Locked() {
		s.t.Error("Locked() = true, want false")
	}
}

func (s *TestSuite) TestFlock_RLocked() {
	if s.flock.RLocked() {
		s.t.Error("RLocked() = true, want false")
	}
}

func (s *TestSuite) TestFlock_String() {
	if got := s.flock.String(); got != s.path {
		s.t.Errorf("String() = %q, want %q", got, s.path)
	}
}

func (s *TestSuite) TestFlock_TryLock() {
	s.checkLockState(s.flock, false, false)

	locked, err := s.flock.TryLock()
	s.requireNoError(err)

	if !locked {
		s.t.Error("TryLock() = false, want true")
	}

	s.checkLockState(s.flock, true, false)

	locked, err = s.flock.TryLock()
	s.requireNoError(err)

	if !locked {
		s.t.Error("second TryLock() = false, want true")
	}

	// make sure we just return false with no error in cases
	// where we would have been blocked
	locked, err = flock.New(s.path, s.opts...).TryLock()
	s.requireNoError(err)

	if locked {
		s.t.Error("contending TryLock() = true, want false")
	}
}

func (s *TestSuite) TestFlock_TryRLock() {
	s.checkLockState(s.flock, false, false)

	locked, err := s.flock.TryRLock()
	s.requireNoError(err)

	if !locked {
		s.t.Error("TryRLock() = false, want true")
	}

	s.checkLockState(s.flock, false, true)

	locked, err = s.flock.TryRLock()
	s.requireNoError(err)

	if !locked {
		s.t.Error("second TryRLock() = false, want true")
	}

	// shared lock should not block.
	flock2 := flock.New(s.path, s.opts...)
	locked, err = flock2.TryRLock()
	s.requireNoError(err)

	switch runtime.GOOS {
	case "aix", "solaris", "illumos":
		// When using POSIX locks, we can't safely read-lock the same
		// inode through two different descriptors at the same time:
		// when the first descriptor is closed, the second descriptor
		// would still be open but silently unlocked. So a second
		// TryRLock must return false.
		if locked {
			s.t.Error("contending TryRLock() = true, want false")
		}
	default:
		if !locked {
			s.t.Error("contending TryRLock() = false, want true")
		}
	}

	// make sure we just return false with no error in cases
	// where we would have been blocked
	_ = s.flock.Unlock()
	_ = flock2.Unlock()
	_ = s.flock.Lock()
	locked, err = flock.New(s.path, s.opts...).TryRLock()
	s.requireNoError(err)

	if locked {
		s.t.Error("TryRLock() against exclusive lock = true, want false")
	}
}

func (s *TestSuite) TestFlock_TryLockContext() {
	ctx, cancel := context.WithCancel(context.Background())

	// happy path
	locked, err := s.flock.TryLockContext(ctx, time.Second)
	s.requireNoError(err)

	if !locked {
		s.t.Error("TryLockContext() = false, want true")
	}

	// context already canceled
	cancel()

	locked, err = flock.New(s.path, s.opts...).TryLockContext(ctx, time.Second)
	if !errors.Is(err, context.Canceled) {
		s.t.Fatalf("TryLockContext() error = %v, want %v", err, context.Canceled)
	}

	if locked {
		s.t.Error("TryLockContext() with canceled context = true, want false")
	}

	// timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	locked, err = flock.New(s.path, s.opts...).TryLockContext(ctx, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		s.t.Fatalf("TryLockContext() error = %v, want %v", err, context.DeadlineExceeded)
	}

	if locked {
		s.t.Error("TryLockContext() after timeout = true, want false")
	}
}

func (s *TestSuite) TestFlock_TryRLockContext() {
	ctx, cancel := context.WithCancel(context.Background())

	// happy path
	locked, err := s.flock.TryRLockContext(ctx, time.Second)
	s.requireNoError(err)

	if !locked {
		s.t.Error("TryRLockContext() = false, want true")
	}

	// context already canceled
	cancel()

	locked, err = flock.New(s.path, s.opts...).TryRLockContext(ctx, time.Second)
	if !errors.Is(err, context.Canceled) {
		s.t.Fatalf("TryRLockContext() error = %v, want %v", err, context.Canceled)
	}

	if locked {
		s.t.Error("TryRLockContext() with canceled context = true, want false")
	}

	// timeout
	_ = s.flock.Unlock()
	_ = s.flock.Lock()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	locked, err = flock.New(s.path, s.opts...).TryRLockContext(ctx, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		s.t.Fatalf("TryRLockContext() error = %v, want %v", err, context.DeadlineExceeded)
	}

	if locked {
		s.t.Error("TryRLockContext() after timeout = true, want false")
	}
}

func (s *TestSuite) TestFlock_Unlock() {
	err := s.flock.Unlock()
	s.requireNoError(err)

	// get a lock for us to unlock
	locked, err := s.flock.TryLock()
	s.requireNoError(err)

	if !locked {
		s.t.Error("TryLock() = false, want true")
	}

	s.checkLockState(s.flock, true, false)

	_, err = os.Stat(s.path)
	if os.IsNotExist(err) {
		s.t.Errorf("lock path does not exist: %v", err)
	}

	err = s.flock.Unlock()
	s.requireNoError(err)
	s.checkLockState(s.flock, false, false)
}

func (s *TestSuite) TestFlock_Lock() {
	s.checkLockState(s.flock, false, false)

	err := s.flock.Lock()
	s.requireNoError(err)
	s.checkLockState(s.flock, true, false)

	// test that the short-circuit works
	err = s.flock.Lock()
	s.requireNoError(err)

	//
	// Test that Lock() is a blocking call
	//
	ch := make(chan error, 2)
	gf := flock.New(s.path, s.opts...)

	defer func() { _ = gf.Unlock() }()

	go func(ch chan<- error) {
		ch <- nil

		ch <- gf.Lock()

		close(ch)
	}(ch)

	errCh, ok := <-ch
	if !ok {
		s.t.Fatal("lock goroutine exited before attempting to acquire the lock")
	}

	s.requireNoError(errCh)

	err = s.flock.Unlock()
	s.requireNoError(err)

	errCh, ok = <-ch
	if !ok {
		s.t.Fatal("lock goroutine exited without reporting a result")
	}

	s.requireNoError(errCh)
	s.checkLockState(s.flock, false, false)
	s.checkLockState(gf, true, false)
}

func (s *TestSuite) TestFlock_RLock() {
	s.checkLockState(s.flock, false, false)

	err := s.flock.RLock()
	s.requireNoError(err)
	s.checkLockState(s.flock, false, true)

	// test that the short-circuit works
	err = s.flock.RLock()
	s.requireNoError(err)

	//
	// Test that RLock() is a blocking call
	//
	ch := make(chan error, 2)

	gf := flock.New(s.path, s.opts...)

	defer func() { _ = gf.Unlock() }()

	go func(ch chan<- error) {
		ch <- nil

		ch <- gf.RLock()

		close(ch)
	}(ch)

	errCh, ok := <-ch
	if !ok {
		s.t.Fatal("lock goroutine exited before attempting to acquire the lock")
	}

	s.requireNoError(errCh)

	err = s.flock.Unlock()
	s.requireNoError(err)

	errCh, ok = <-ch
	if !ok {
		s.t.Fatal("lock goroutine exited without reporting a result")
	}

	s.requireNoError(errCh)
	s.checkLockState(s.flock, false, false)
	s.checkLockState(gf, false, true)
}

func (s *TestSuite) TestFlock_Stat() {
	// Test Stat when file doesn't exist yet (for non-directory case)
	if !s.dir {
		_, err := s.flock.Stat()
		if !os.IsNotExist(err) {
			s.t.Errorf("Stat() error = %v, want an os.ErrNotExist error", err)
		}
	}

	// Create the lock file
	locked, err := s.flock.TryLock()
	s.requireNoError(err)

	if !locked {
		s.t.Error("TryLock() = false, want true")
	}

	// Test Stat after lock is acquired
	info, err := s.flock.Stat()
	s.requireNoError(err)

	if info == nil {
		s.t.Fatal("Stat() returned nil FileInfo")
	}

	// Check modification time is recent
	modTime := info.ModTime()
	s.WithinDuration(time.Now(), modTime, 1*time.Second)

	// Unlock and verify Stat still works (file persists)
	err = s.flock.Unlock()
	s.requireNoError(err)

	info, err = s.flock.Stat()
	s.requireNoError(err)

	if info == nil {
		s.t.Fatal("Stat() returned nil FileInfo")
	}

	// The modification time should be approximately the same as before
	s.WithinDuration(modTime, info.ModTime(), 100*time.Millisecond)
}

func (s *TestSuite) WithinDuration(want, got time.Time, delta time.Duration) {
	s.t.Helper()

	difference := got.Sub(want)
	if difference < -delta || difference > delta {
		s.t.Errorf("time difference = %v, want within %v (got %v, want %v)", difference, delta, got, want)
	}
}

func (s *TestSuite) checkLockState(f *flock.Flock, locked, rlocked bool) {
	s.t.Helper()

	if got := f.Locked(); got != locked {
		s.t.Errorf("Locked() = %v, want %v", got, locked)
	}

	if got := f.RLocked(); got != rlocked {
		s.t.Errorf("RLocked() = %v, want %v", got, rlocked)
	}
}

func (s *TestSuite) requireNoError(err error) {
	s.t.Helper()

	if err != nil {
		s.t.Fatal(err)
	}
}
