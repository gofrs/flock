// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2024 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package flock_test

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite

	path  string
	flock *flock.Flock
}

func Test(t *testing.T) { suite.Run(t, &TestSuite{}) }

func (s *TestSuite) SetupTest() {
	tmpFile, err := os.CreateTemp(os.TempDir(), "go-flock-")
	s.Require().NoError(err)

	s.Require().NotNil(tmpFile)

	s.path = tmpFile.Name()

	defer os.Remove(s.path)
	tmpFile.Close()

	s.flock = flock.New(s.path)
}

func (s *TestSuite) TearDownTest() {
	_ = s.flock.Unlock()
	os.Remove(s.path)
}

func (s *TestSuite) TestNew() {
	f := flock.New(s.path)
	s.Require().NotNil(f)

	s.Equal(s.path, f.Path())
	s.False(f.Locked())
	s.False(f.RLocked())
}

func (s *TestSuite) TestFlock_Path() {
	path := s.flock.Path()
	s.Equal(s.path, path)
}

func (s *TestSuite) TestFlock_Locked() {
	locked := s.flock.Locked()
	s.False(locked)
}

func (s *TestSuite) TestFlock_RLocked() {
	locked := s.flock.RLocked()
	s.False(locked)
}

func (s *TestSuite) TestFlock_String() {
	str := s.flock.String()
	s.Equal(s.path, str)
}

func (s *TestSuite) TestFlock_TryLock() {
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())

	var locked bool
	var err error

	locked, err = s.flock.TryLock()
	s.Require().NoError(err)
	s.True(locked)
	s.True(s.flock.Locked())
	s.False(s.flock.RLocked())

	locked, err = s.flock.TryLock()
	s.Require().NoError(err)
	s.True(locked)

	// make sure we just return false with no error in cases
	// where we would have been blocked
	locked, err = flock.New(s.path).TryLock()
	s.Require().NoError(err)
	s.False(locked)
}

func (s *TestSuite) TestFlock_TryRLock() {
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())

	var locked bool
	var err error

	locked, err = s.flock.TryRLock()
	s.Require().NoError(err)
	s.True(locked)
	s.False(s.flock.Locked())
	s.True(s.flock.RLocked())

	locked, err = s.flock.TryRLock()
	s.Require().NoError(err)
	s.True(locked)

	// shared lock should not block.
	flock2 := flock.New(s.path)
	locked, err = flock2.TryRLock()
	s.Require().NoError(err)

	if runtime.GOOS == "aix" {
		// When using POSIX locks, we can't safely read-lock the same
		// inode through two different descriptors at the same time:
		// when the first descriptor is closed, the second descriptor
		// would still be open but silently unlocked. So a second
		// TryRLock must return false.
		s.False(locked)
	} else {
		s.True(locked)
	}

	// make sure we just return false with no error in cases
	// where we would have been blocked
	_ = s.flock.Unlock()
	_ = flock2.Unlock()
	_ = s.flock.Lock()
	locked, err = flock.New(s.path).TryRLock()
	s.Require().NoError(err)
	s.False(locked)
}

func (s *TestSuite) TestFlock_TryLockContext() {
	// happy path
	ctx, cancel := context.WithCancel(context.Background())
	locked, err := s.flock.TryLockContext(ctx, time.Second)
	s.Require().NoError(err)
	s.True(locked)

	// context already canceled
	cancel()

	locked, err = flock.New(s.path).TryLockContext(ctx, time.Second)
	s.Require().ErrorIs(err, context.Canceled)
	s.False(locked)

	// timeout
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	locked, err = flock.New(s.path).TryLockContext(ctx, time.Second)
	s.Require().ErrorIs(err, context.DeadlineExceeded)
	s.False(locked)
}

func (s *TestSuite) TestFlock_TryRLockContext() {
	// happy path
	ctx, cancel := context.WithCancel(context.Background())
	locked, err := s.flock.TryRLockContext(ctx, time.Second)
	s.Require().NoError(err)
	s.True(locked)

	// context already canceled
	cancel()
	locked, err = flock.New(s.path).TryRLockContext(ctx, time.Second)
	s.Require().ErrorIs(err, context.Canceled)
	s.False(locked)

	// timeout
	_ = s.flock.Unlock()
	_ = s.flock.Lock()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	locked, err = flock.New(s.path).TryRLockContext(ctx, time.Second)
	s.Require().ErrorIs(err, context.DeadlineExceeded)
	s.False(locked)
}

func (s *TestSuite) TestFlock_Unlock() {
	var err error

	err = s.flock.Unlock()
	s.Require().NoError(err)

	// get a lock for us to unlock
	locked, err := s.flock.TryLock()
	s.Require().NoError(err)
	s.True(locked)
	s.True(s.flock.Locked())
	s.False(s.flock.RLocked())

	_, err = os.Stat(s.path)
	s.False(os.IsNotExist(err))

	err = s.flock.Unlock()
	s.Require().NoError(err)
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())
}

func (s *TestSuite) TestFlock_Lock() {
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())

	var err error

	err = s.flock.Lock()
	s.Require().NoError(err)
	s.True(s.flock.Locked())
	s.False(s.flock.RLocked())

	// test that the short-circuit works
	err = s.flock.Lock()
	s.Require().NoError(err)

	//
	// Test that Lock() is a blocking call
	//
	ch := make(chan error, 2)
	gf := flock.New(s.path)
	defer func() { _ = gf.Unlock() }()

	go func(ch chan<- error) {
		ch <- nil
		ch <- gf.Lock()
		close(ch)
	}(ch)

	errCh, ok := <-ch
	s.True(ok)
	s.Require().NoError(errCh)

	err = s.flock.Unlock()
	s.Require().NoError(err)

	errCh, ok = <-ch
	s.True(ok)
	s.Require().NoError(errCh)
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())
	s.True(gf.Locked())
	s.False(gf.RLocked())
}

func (s *TestSuite) TestFlock_RLock() {
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())

	var err error

	err = s.flock.RLock()
	s.Require().NoError(err)
	s.False(s.flock.Locked())
	s.True(s.flock.RLocked())

	// test that the short-circuit works
	err = s.flock.RLock()
	s.Require().NoError(err)

	//
	// Test that RLock() is a blocking call
	//
	ch := make(chan error, 2)
	gf := flock.New(s.path)
	defer func() { _ = gf.Unlock() }()

	go func(ch chan<- error) {
		ch <- nil
		ch <- gf.RLock()
		close(ch)
	}(ch)

	errCh, ok := <-ch
	s.True(ok)
	s.Require().NoError(errCh)

	err = s.flock.Unlock()
	s.Require().NoError(err)

	errCh, ok = <-ch
	s.True(ok)
	s.Require().NoError(errCh)
	s.False(s.flock.Locked())
	s.False(s.flock.RLocked())
	s.False(gf.Locked())
	s.True(gf.RLocked())
}
