package ui

import "testing"

// The expected values are pinned against the Node implementation:
//
//	node -e 'v => v.replace(/\S(?=\S{4})/g, "*")'
func TestMask(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"a", "a"},
		{"ab", "ab"},
		{"abcd", "abcd"},
		{"abcde", "*bcde"},
		{"abcdefgh", "****efgh"},
		{"my secret value", "my **cret *alue"},
		{"p@ss word1234", "p@ss ****1234"},
		{"ab cd ef ghij", "ab cd ef ghij"},
		{"1234567890", "******7890"},
	}
	for _, c := range cases {
		if got := Mask(c.in); got != c.want {
			t.Errorf("Mask(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
