package clierr

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorFormatting(t *testing.T) {
	if got := User("message", "hint").Error(); got != "message (hint)" {
		t.Errorf("with hint: %q", got)
	}
	if got := User("message", "").Error(); got != "message" {
		t.Errorf("without hint: %q", got)
	}
	if got := UserT("Title", "message", "").Error(); got != "message" {
		t.Errorf("title must not appear in Error(): %q", got)
	}
	if got := Silent().Error(); got != "" {
		t.Errorf("Silent must have an empty message: %q", got)
	}
}

func TestIsUserSeesThroughWrapping(t *testing.T) {
	base := User("expected", "")
	wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", base))
	if !IsUser(wrapped) {
		t.Errorf("IsUser must see through %%w chains")
	}
	if IsUser(errors.New("plain")) {
		t.Errorf("plain errors are not UserErrors")
	}
	if IsUser(nil) {
		t.Errorf("nil is not a UserError")
	}
}

func TestWrapPreservesCause(t *testing.T) {
	cause := errors.New("root cause")
	err := WrapT(cause, "Title", "friendly", "hint")
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is must reach the cause through Unwrap")
	}
	var ue *UserError
	if !errors.As(err, &ue) || ue.Title != "Title" || ue.Hint != "hint" {
		t.Errorf("As mismatch: %+v", ue)
	}
}
