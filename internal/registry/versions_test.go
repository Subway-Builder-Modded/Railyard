package registry

import (
	"testing"

	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

func TestFilterSemverVersions(t *testing.T) {
	reg := NewRegistry(testLogSink{})
	filtered := reg.filterSemverVersions([]types.VersionInfo{
		{Version: "1.2.3"},
		{Version: "v2.3.4"},
		{Version: "1.2"},
		{Version: "1.2.3-beta.1"},
		{Version: "1.2.3+build.1"},
		{Version: "not-semver"},
		{Version: ""},
	}, "test")

	require.Len(t, filtered, 2)
	require.Equal(t, "1.2.3", filtered[0].Version)
	require.Equal(t, "v2.3.4", filtered[1].Version)
}
