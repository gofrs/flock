// Copyright 2015 Tim Heckman. All rights reserved.
// Copyright 2018-2024 The Gofrs. All rights reserved.
// Use of this source code is governed by the BSD 3-Clause
// license that can be found in the LICENSE file.

package flock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "go-flock-")
	require.NoError(t, err)

	_ = tmpFile.Close()
	_ = os.Remove(tmpFile.Name())

	lock := New(tmpFile.Name())

	locked, err := lock.TryLock()
	require.NoError(t, err)
	require.True(t, locked)

	newLock := New(tmpFile.Name())

	locked, err = newLock.TryLock()
	require.NoError(t, err)
	require.False(t, locked)

	assert.Nil(t, newLock.fh, "file handle should have been released and be nil")

	err = lock.Unlock()
	require.NoError(t, err)
}
