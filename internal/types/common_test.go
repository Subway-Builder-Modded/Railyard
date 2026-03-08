package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidAssetType(t *testing.T) {
	require.True(t, IsValidAssetType(AssetTypeMap))
	require.True(t, IsValidAssetType(AssetTypeMod))
	require.False(t, IsValidAssetType(AssetType("unknown")))
}

func TestIsValidSemverVersion(t *testing.T) {
	require.True(t, IsValidSemverVersion(Version("1.2.3")))
	require.True(t, IsValidSemverVersion(Version("v1.2.3")))
	require.True(t, IsValidSemverVersion(Version(" 1.2.3 ")))

	require.False(t, IsValidSemverVersion(Version("1.2")))
	require.False(t, IsValidSemverVersion(Version("1.2.3.4")))
	require.False(t, IsValidSemverVersion(Version("1.2.3-beta.1")))
	require.False(t, IsValidSemverVersion(Version("1.2.3+build.9")))
	require.False(t, IsValidSemverVersion(Version("not-semver")))
	require.False(t, IsValidSemverVersion(Version("")))
}
