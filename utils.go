package main

import "golang.org/x/exp/slices"

func Filter[S ~[]E, E any](s S, match func(E) bool) S {
	compacted := slices.CompactFunc(s, func(value, last E) bool {
		isMatch := match(value)
		return isMatch
	})
	clipped := slices.Clip(compacted)
	return clipped
}

func Count[S ~[]E, E any](s S, match func(E) bool) int {
	count := 0
	for _, value := range s {
		if match(value) {
			count++
		}
	}
	return count
}
