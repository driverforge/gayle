// Package clierr defines UserError: an expected, anticipated condition that
// carries a human-friendly message (e.g. a missing gayle.yml, or a required
// config that has no value yet).
//
// The top-level handler renders a UserError as a friendly message and exits 1.
// Any error that reaches the top WITHOUT being a UserError is treated as
// unexpected (a crash): it renders a crash card and exits 2. So the rule is
// simply "if you anticipated it, wrap it in a UserError."
//
// # Construction
//
// Prefer the constructors over a struct literal so the common cases read the
// same everywhere:
//
//   - [User] / [UserT]  — a fresh message (UserT adds a title heading).
//   - [Wrap] / [WrapT]  — same, but carrying an underlying cause for errors.Is/As.
//   - [Silent]          — exit non-zero, render nothing (help-then-fail paths).
//
// Reach for a &UserError{…} literal only when a call needs a shape the helpers
// don't cover (e.g. setting Err without a title, or building the value across
// several statements). Don't build with a helper and then mutate .Title after
// the fact — use UserT/WrapT instead.
//
// # Sentinels
//
// Two sentinel idioms coexist, deliberately:
//
//   - A *UserError sentinel is already friendly and can be returned as-is;
//     errors.Is matches it and the renderer shows its card.
//   - A plain sentinel (e.g. a package's ErrNotFound) is an internal signal that
//     the command layer translates into a UserError at the call site, so the same
//     condition can read differently depending on the command that hit it.
package clierr

import "errors"

// UserError is an expected, user-facing error. Title is an optional short label
// for the kind of problem (e.g. "Invalid usage"); Message is the friendly
// headline; Hint is an optional actionable next step (e.g. "run `gayle generate`");
// Err is an optional underlying cause kept for errors.Is/As unwrapping (it is
// never shown to the user).
type UserError struct {
	Title   string
	Message string
	Hint    string
	Err     error
}

// Error renders the message with the hint appended in parentheses, so a
// UserError still reads sensibly in plain (non-TTY) output and logs.
func (e *UserError) Error() string {
	if e.Hint != "" {
		return e.Message + " (" + e.Hint + ")"
	}
	return e.Message
}

// Unwrap exposes the underlying cause to errors.Is/errors.As.
func (e *UserError) Unwrap() error { return e.Err }

// User builds a UserError with a friendly message and an optional hint.
func User(message, hint string) *UserError {
	return &UserError{Message: message, Hint: hint}
}

// UserT builds a UserError with a title heading, message and optional hint — the
// titled counterpart to User, so callers needn't set .Title after construction.
func UserT(title, message, hint string) *UserError {
	return &UserError{Title: title, Message: message, Hint: hint}
}

// Silent is an expected error that should exit non-zero but render nothing — for
// paths that have already printed their own output (e.g. help shown for a bare
// `gayle` invocation, which must still fail).
func Silent() *UserError {
	return &UserError{}
}

// Wrap builds a UserError that also carries an underlying cause, so callers can
// still match the original error with errors.Is while presenting a friendly face.
func Wrap(cause error, message, hint string) *UserError {
	return &UserError{Message: message, Hint: hint, Err: cause}
}

// WrapT is Wrap with a title heading — for a titled card that also needs to carry
// an underlying cause for errors.Is/As.
func WrapT(cause error, title, message, hint string) *UserError {
	return &UserError{Title: title, Message: message, Hint: hint, Err: cause}
}

// IsUser reports whether err is, or wraps, a UserError.
func IsUser(err error) bool {
	var ue *UserError
	return errors.As(err, &ue)
}
