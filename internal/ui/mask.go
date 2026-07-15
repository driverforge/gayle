package ui

import "unicode"

// Mask hides a secret value the way the Node CLI did — the regex
// /\S(?=\S{4})/g: every non-space character that is immediately followed by
// four non-space characters becomes '*'. For a plain token that means
// "everything but the last four characters"; values of four characters or
// fewer are left as-is, and characters adjacent to whitespace unmask early.
// Quirky, but pipelines and habits are calibrated to it.
func Mask(value string) string {
	r := []rune(value)
	out := make([]rune, len(r))
	for i, c := range r {
		if !unicode.IsSpace(c) && i+4 < len(r) && allNonSpace(r[i+1:i+5]) {
			out[i] = '*'
		} else {
			out[i] = c
		}
	}
	return string(out)
}

func allNonSpace(rs []rune) bool {
	for _, c := range rs {
		if unicode.IsSpace(c) {
			return false
		}
	}
	return true
}
