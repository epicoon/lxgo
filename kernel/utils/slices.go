package utils

// Returns a new slice containing all elements from slice1
// that are not present in slice2.
func SliceDiff[T comparable](slice1, slice2 []T) []T {
	diff := []T{}
	// Create a map to store elements of slice2 for efficient lookup
	inSlice2 := make(map[T]bool)
	for _, item := range slice2 {
		inSlice2[item] = true
	}
	// Iterate through slice1 and add elements not found in slice2 to diff
	for _, item := range slice1 {
		if !inSlice2[item] {
			diff = append(diff, item)
		}
	}
	return diff
}
