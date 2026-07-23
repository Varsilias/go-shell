package utils

import "testing"

func TestLongestCommonPrefix(t *testing.T) {
	testCases := []struct {
		name  string
		input []string
		want  string
	}{
		{
			name:  "returns empty string when word list is empty",
			input: []string{},
			want:  "",
		},
		{
			name:  "returns sole word when word list element has a single element",
			input: []string{"foo"},
			want:  "foo",
		},
		{
			name:  "returns the longest common prefix when known",
			input: []string{"xyz_foo_bar", "xyz_foo_baz", "xyz_fox"},
			want:  "xyz_fo",
		},
		{
			name:  "returns the longest common prefix when known2",
			input: []string{"flower", "flow", "flight"},
			want:  "fl",
		},
		{
			name:  "returns empty string when there is no possible LCP",
			input: []string{"dog", "racecar", "car"},
			want:  "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			prefix := LongestCommonPrefix(tt.input)
			if prefix != tt.want {
				t.Errorf("got %s, expected %s", prefix, tt.want)
			}
		})
	}
}
