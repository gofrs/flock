// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2024 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

//go:build windows

package flock

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Use of 0x00000000 for the shared lock is a guess based on some the MS Windows `LockFileEX` docs,
// which document the `LOCKFILE_EXCLUSIVE_LOCK` flag as:
//
// > The function requests an exclusive lock. Otherwise, it requests a shared lock.
//
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx

const (
	winLockfileSharedLock      = 0x00000000
	winLockfileFailImmediately = 0x00000001
	winLockfileExclusiveLock   = 0x00000002
)

var modkernel32 = windows.NewLazySystemDLL("kernel32.dll")

var (
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

//nolint:unparam
func lockFileEx(handle syscall.Handle, flags, reserved, numberOfBytesToLockLow, numberOfBytesToLockHigh uint32, offset *syscall.Overlapped) (bool, syscall.Errno) {
	r1, _, errNo := syscall.SyscallN(
		procLockFileEx.Addr(),
		uintptr(handle),
		uintptr(flags),
		uintptr(reserved),
		uintptr(numberOfBytesToLockLow),
		uintptr(numberOfBytesToLockHigh),
		uintptr(unsafe.Pointer(offset)))

	if r1 == 1 {
		return true, 0
	}

	if errNo == 0 {
		return false, syscall.EINVAL
	}

	return false, errNo
}

func unlockFileEx(handle syscall.Handle, reserved, numberOfBytesToLockLow, numberOfBytesToLockHigh uint32, offset *syscall.Overlapped) (bool, syscall.Errno) {
	r1, _, errNo := syscall.SyscallN(
		procUnlockFileEx.Addr(),
		uintptr(handle),
		uintptr(reserved),
		uintptr(numberOfBytesToLockLow),
		uintptr(numberOfBytesToLockHigh),
		uintptr(unsafe.Pointer(offset)))

	if r1 == 1 {
		return true, 0
	}

	if errNo == 0 {
		return false, syscall.EINVAL
	}

	return false, errNo
}
