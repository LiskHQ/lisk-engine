// Package strings provides utility functions for string slices.
package strings

import "math/rand"

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// Contain return true if the strings includes at least one target string.
func Contain(strings []string, target string) bool {
	for _, str := range strings {
		if str == target {
			return true
		}
	}
	return false
}

// Unique removes duplicate elements from the list and return a new slice.
func Unique(list []string) []string {
	temp := map[string]bool{}
	for _, val := range list {
		temp[val] = true
	}
	result := []string{}
	for key := range temp {
		result = append(result, key)
	}
	return result
}

// IsUnique return true if the elements of input list are unique.
func IsUnique(list []string) bool {
	temp := Unique(list)
	return len(temp) == len(list)
}

// GenerateRandom return random string with n length.
func GenerateRandom(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
