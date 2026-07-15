package buildinfo

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	got := String()
	if !strings.HasPrefix(got, "gayle ") ||
		!strings.Contains(got, Version) ||
		!strings.Contains(got, Commit) ||
		!strings.Contains(got, Date) {
		t.Errorf("String() = %q", got)
	}
}
