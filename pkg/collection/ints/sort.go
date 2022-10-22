package ints

import "sort"

// MaxUInt32 returns max number within input (input order will be mutated).
func Max[T Integer](nums ...T) T {
	if len(nums) == 0 {
		panic("Cannot determine max with empty slice")
	}
	// sort number desc
	sorting := make([]T, len(nums))
	copy(sorting, nums)
	sort.Slice(sorting, func(i, j int) bool {
		return sorting[i] > sorting[j]
	})
	return sorting[0]
}

// MaxUInt32 returns max number within input (input order will be mutated).
func Min[T Integer](nums ...T) T {
	if len(nums) == 0 {
		panic("Cannot determine min with empty slice")
	}
	// sort number desc
	sorting := make([]T, len(nums))
	copy(sorting, nums)
	sort.Slice(sorting, func(i, j int) bool {
		return sorting[i] < sorting[j]
	})
	return sorting[0]
}
