package paramstore

import (
	"errors"
	"strings"
	"testing"
)

func TestShortKey(t *testing.T) {
	cases := []struct{ name, want string }{
		{"/dev/config/DB_NAME", "DB_NAME"},
		{"graph/DB_NAME", "DB_NAME"},
		{"BARE", "BARE"},
		{"/trailing/", ""},
	}
	for _, c := range cases {
		if got := ShortKey(c.name); got != c.want {
			t.Errorf("ShortKey(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestShortKeysDropsEmpty(t *testing.T) {
	got := ShortKeys(map[string]string{
		"/dev/config/DB_NAME": "db",
		"/dev/config/":        "dropped", // empty short key vanishes (Node parity)
	})
	if len(got) != 1 || got["DB_NAME"] != "db" {
		t.Errorf("ShortKeys = %v", got)
	}
}

func TestKeyErrorsFormatting(t *testing.T) {
	one := KeyErrors{{Key: "/dev/config/A", Err: errors.New("denied")}}
	if got := one.Error(); got != "1 parameter failed: /dev/config/A: denied" {
		t.Errorf("single: %q", got)
	}

	two := KeyErrors{
		{Key: "/dev/config/A", Err: errors.New("denied")},
		{Key: "/dev/config/B", Err: errors.New("throttled")},
	}
	got := two.Error()
	if !strings.HasPrefix(got, "2 parameters failed:") ||
		!strings.Contains(got, "/dev/config/A: denied") ||
		!strings.Contains(got, "/dev/config/B: throttled") {
		t.Errorf("multi must name every key: %q", got)
	}
}

func TestKeyErrorsOrNil(t *testing.T) {
	var empty KeyErrors
	if empty.OrNil() != nil {
		t.Errorf("empty KeyErrors must be nil error")
	}
	nonEmpty := KeyErrors{{Key: "k", Err: errors.New("x")}}
	if nonEmpty.OrNil() == nil {
		t.Errorf("non-empty KeyErrors must be an error")
	}
}
