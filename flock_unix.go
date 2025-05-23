// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2025 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package flock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// Lock is a blocking call to try and take an exclusive file lock.
// It will wait until it is able to obtain the exclusive file lock.
// It's recommended that TryLock() be used over this function.
// This function may block the ability to query the current Locked() or RLocked() status due to a RW-mutex lock.
//
// If we are already exclusive-locked,
// this function short-circuits and returns immediately assuming it can take the mutex lock.
//
// If the *Flock has a shared lock (RLock),
// this may transparently replace the shared lock with an exclusive lock on some UNIX-like operating systems.
// Be careful when using exclusive locks in conjunction with shared locks (RLock()),
// because calling Unlock() may accidentally release the exclusive lock that was once a shared lock.
func (f *Flock) Lock() error {
	return f.lock(&f.l, unix.LOCK_EX)
}

// RLock is a blocking call to try and take a shared file lock.
// It will wait until it is able to obtain the shared file lock.
// It's recommended that TryRLock() be used over this function.
// This function may block the ability to query the current Locked() or RLocked() status due to a RW-mutex lock.
//
// If we are already shared-locked,
// this function short-circuits and returns immediately assuming it can take the mutex lock.
func (f *Flock) RLock() error {
	return f.lock(&f.r, unix.LOCK_SH)
}

func (f *Flock) lock(locked *bool, flag int) error {
	f.m.Lock()
	defer f.m.Unlock()

	if *locked {
		return nil
	}

	if f.fh == nil {
		if err := f.setFh(f.flag); err != nil {
			return err
		}

		defer f.ensureFhState()
	}

	err := unix.Flock(int(f.fh.Fd()), flag)
	if err != nil {
		shouldRetry, reopenErr := f.reopenFDOnError(err)
		if reopenErr != nil {
			return reopenErr
		}

		if !shouldRetry {
			return err
		}

		err = unix.Flock(int(f.fh.Fd()), flag)
		if err != nil {
			return err
		}
	}

	*locked = true

	return nil
}

// Unlock is a function to unlock the file.
// This file takes a RW-mutex lock,
// so while it is running the Locked() and RLocked() functions will be blocked.
//
// This function short-circuits if we are unlocked already.
// If not, it calls unix.LOCK_UN on the file and closes the file descriptor.
// It does not remove the file from disk. It's up to your application to do.
//
// Please note,
// if your shared lock became an exclusive lock,
// this may unintentionally drop the exclusive lock if called by the consumer that believes they have a shared lock.
// Please see Lock() for more details.
func (f *Flock) Unlock() error {
	f.m.Lock()
	defer f.m.Unlock()

	// If we aren't locked or if the lockfile instance is nil
	// just return a nil error because we are unlocked.
	if (!f.l && !f.r) || f.fh == nil {
		return nil
	}

	// Mark the file as unlocked.
	err := unix.Flock(int(f.fh.Fd()), unix.LOCK_UN)
	if err != nil {
		return err
	}

	f.reset()

	return nil
}

// TryLock is the preferred function for taking an exclusive file lock.
// This function takes an RW-mutex lock before it tries to lock the file,
// so there is the possibility that this function may block for a short time
// if another goroutine is trying to take any action.
//
// The actual file lock is non-blocking.
// If we are unable to get the exclusive file lock,
// the function will return false instead of waiting for the lock.
// If we get the lock, we also set the *Flock instance as being exclusive-locked.
func (f *Flock) TryLock() (bool, error) {
	return f.try(&f.l, unix.LOCK_EX)
}

// TryRLock is the preferred function for taking a shared file lock.
// This function takes an RW-mutex lock before it tries to lock the file,
// so there is the possibility that this function may block for a short time
// if another goroutine is trying to take any action.
//
// The actual file lock is non-blocking.
// If we are unable to get the shared file lock,
// the function will return false instead of waiting for the lock.
// If we get the lock, we also set the *Flock instance as being share-locked.
func (f *Flock) TryRLock() (bool, error) {
	return f.try(&f.r, unix.LOCK_SH)
}

func (f *Flock) try(locked *bool, flag int) (bool, error) {
	f.m.Lock()
	defer f.m.Unlock()

	if *locked {
		return true, nil
	}

	if f.fh == nil {
		if err := f.setFh(f.flag); err != nil {
			return false, err
		}

		defer f.ensureFhState()
	}

	var retried bool
retry:
	err := unix.Flock(int(f.fh.Fd()), flag|unix.LOCK_NB)

	switch {
	case errors.Is(err, unix.EWOULDBLOCK):
		return false, nil
	case err == nil:
		*locked = true
		return true, nil
	}

	if !retried {
		shouldRetry, reopenErr := f.reopenFDOnError(err)
		if reopenErr != nil {
			return false, reopenErr
		} else if shouldRetry {
			retried = true
			goto retry
		}
	}

	return false, err
}

// reopenFDOnError determines whether we should reopen the file handle in readwrite mode and try again.
// This comes from `util-linux/sys-utils/flock.c`:
// > Since Linux 3.4 (commit 55725513)
// > Probably NFSv4 where flock() is emulated by fcntl().
// > https://github.com/util-linux/util-linux/blob/198e920aa24743ef6ace4e07cf6237de527f9261/sys-utils/flock.c#L374-L390
func (f *Flock) reopenFDOnError(err error) (bool, error) {
	if !errors.Is(err, unix.EIO) && !errors.Is(err, unix.EBADF) {
		return false, nil
	}

	st, err := f.fh.Stat()
	if err != nil {
		return false, nil
	}

	if st.Mode()&f.perm != f.perm {
		return false, nil
	}

	f.resetFh()

	// reopen in read-write mode and set the file handle
	err = f.setFh(f.flag | os.O_RDWR)
	if err != nil {
		return false, err
	}

	return true, nil
}
