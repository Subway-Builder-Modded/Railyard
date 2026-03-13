package lock

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObtainLockSecondInstanceReturnsAlreadyRunning(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	first, alreadyRunning, err := Acquire()
	require.NoError(t, err)
	require.False(t, alreadyRunning)
	require.NotNil(t, first)
	defer func() {
		require.NoError(t, first.Release())
	}()

	second, alreadyRunning, err := Acquire()
	require.NoError(t, err)
	require.True(t, alreadyRunning)
	require.Nil(t, second)
}

func TestObtainLockReleaseThenReacquire(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	first, alreadyRunning, err := Acquire()
	require.NoError(t, err)
	require.False(t, alreadyRunning)
	require.NotNil(t, first)

	require.NoError(t, first.Release())

	second, alreadyRunning, err := Acquire()
	require.NoError(t, err)
	require.False(t, alreadyRunning)
	require.NotNil(t, second)
	defer func() {
		require.NoError(t, second.Release())
	}()
}

func TestReleaseIdempotent(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	lock, alreadyRunning, err := Acquire()
	require.NoError(t, err)
	require.False(t, alreadyRunning)
	require.NotNil(t, lock)

	require.NoError(t, lock.Release())
	require.NoError(t, lock.Release())
}
