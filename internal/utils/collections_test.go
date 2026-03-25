package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrEmptyMapReturnsEmptyMapForNil(t *testing.T) {
	var input map[string]int
	output := OrEmptyMap(input)

	require.NotNil(t, output)
	require.Equal(t, 0, len(output))
}

func TestOrEmptyMapReturnsOriginalMapWhenNonNil(t *testing.T) {
	input := map[string]int{"a": 1}
	output := OrEmptyMap(input)

	require.Equal(t, input, output)
}

func TestOrEmptySliceReturnsEmptySliceForNil(t *testing.T) {
	var input []string
	output := OrEmptySlice(input)

	require.NotNil(t, output)
	require.Equal(t, 0, len(output))
}

func TestOrEmptySliceReturnsOriginalSliceWhenNonNil(t *testing.T) {
	input := []string{"a"}
	output := OrEmptySlice(input)

	require.Equal(t, input, output)
}

func TestCloneMapReturnsEmptyMapForNil(t *testing.T) {
	var input map[string]int
	output := CloneMap(input)

	require.NotNil(t, output)
	require.Empty(t, output)
}

func TestCloneMapReturnsCopyForNonNil(t *testing.T) {
	input := map[string]int{"a": 1}
	output := CloneMap(input)

	require.Equal(t, input, output)

	output["a"] = 2
	require.Equal(t, 1, input["a"])
}

func TestOrEmptyNestedMapReturnsEmptyMapForNil(t *testing.T) {
	var input map[string]map[string]int
	output := OrEmptyNestedMap(input)

	require.NotNil(t, output)
	require.Empty(t, output)
}

func TestOrEmptyNestedMapReturnsOriginalMapWhenNonNil(t *testing.T) {
	input := map[string]map[string]int{"a": {"b": 1}}
	output := OrEmptyNestedMap(input)

	require.Equal(t, input, output)
}

func TestCloneNestedMapReturnsDeepCopy(t *testing.T) {
	input := map[string]map[string]int{
		"a": {"x": 1},
	}
	output := CloneNestedMap(input)

	require.Equal(t, input, output)

	output["a"]["x"] = 2
	require.Equal(t, 1, input["a"]["x"])
}

func TestMapEqual(t *testing.T) {
	require.True(t, MapEqual(map[string]string{"a": "1"}, map[string]string{"a": "1"}))
	require.False(t, MapEqual(map[string]string{"a": "1"}, map[string]string{"a": "2"}))
	require.False(t, MapEqual(map[string]string{"a": "1"}, map[string]string{"a": "1", "b": "2"}))
}

func TestCloneStringMap(t *testing.T) {
	input := map[string]string{"a": "1"}
	output := CloneStringMap(input)
	require.Equal(t, input, output)

	output["a"] = "2"
	require.Equal(t, "1", input["a"])
}
