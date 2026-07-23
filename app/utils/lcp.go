package utils

import "strings"

// LongestCommonPrefix takes a list of words that match a starting character
// and returns the Longest Common prefix between the words
// E.G:
// words = ["xyz_foo_bar", "xyz_foo_baz", "xyz_fox"]
// return xyz_fo
func LongestCommonPrefix(words []string) string {
	var b strings.Builder
	if len(words) == 0 {
		return b.String()
	}

	fw := words[0]

	for i := range len(fw) {
		for _, word := range words {
			if i == len(word) || word[i] != fw[i] {
				return b.String()
			}
		}
		b.WriteByte(fw[i])
	}

	return b.String()
}
