package ui

import (
	"errors"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/driverforge/gayle/internal/clierr"
)

// ExpectedIfAborted maps an interactive abort — Ctrl-C or Esc out of a huh
// form, or a killed/interrupted Bubble Tea program — to an expected
// clierr.UserError, so it renders as a friendly "Cancelled" notice (and still
// exits non-zero: an aborted run wrote nothing). Any other error (or nil)
// passes through unchanged.
func ExpectedIfAborted(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) ||
		errors.Is(err, tea.ErrProgramKilled) ||
		errors.Is(err, tea.ErrInterrupted) {
		return clierr.WrapT(err, "Cancelled", "No changes were made.", "")
	}
	return err
}
