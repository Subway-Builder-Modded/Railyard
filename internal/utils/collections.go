package utils

// OrEmptyMap returns an empty map when the input map is nil.
func OrEmptyMap[K comparable, V any](value map[K]V) map[K]V {
	if value == nil {
		return map[K]V{}
	}
	return value
}

// OrEmptySlice returns an empty slice when the input slice is nil.
func OrEmptySlice[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}

// OrEmptyNestedMap returns an empty 2-level map when the input map is nil.
func OrEmptyNestedMap[K1 comparable, K2 comparable, V any](value map[K1]map[K2]V) map[K1]map[K2]V {
	if value == nil {
		return map[K1]map[K2]V{}
	}
	return value
}

// CloneMap returns a shallow copy of the input map.
// When input is nil, it returns an empty (non-nil) map.
func CloneMap[K comparable, V any](value map[K]V) map[K]V {
	cloned := make(map[K]V, len(value))
	for k, v := range value {
		cloned[k] = v
	}
	return cloned
}

// CloneNestedMap returns a deep copy of a 2-level map.
// When input is nil, it returns an empty (non-nil) map.
func CloneNestedMap[K1 comparable, K2 comparable, V any](value map[K1]map[K2]V) map[K1]map[K2]V {
	cloned := make(map[K1]map[K2]V, len(value))
	for k, inner := range value {
		cloned[k] = CloneMap(inner)
	}
	return cloned
}

// CloneStringMap returns a shallow copy of map[string]string.
func CloneStringMap(value map[string]string) map[string]string {
	return CloneMap(value)
}

// MapEqual returns true when both maps contain exactly the same keys and values.
func MapEqual[K comparable, V comparable](left map[K]V, right map[K]V) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		rightValue, ok := right[key]
		if !ok || rightValue != value {
			return false
		}
	}
	return true
}
