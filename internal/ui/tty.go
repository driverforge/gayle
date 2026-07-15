package ui

import (
	"os"

	"golang.org/x/term"
)

// Interactive reports whether interactive prompting is possible: not running
// under CI, and both stdin and stdout are terminals. `gayle run -i` fails fast
// when this is false — the Node version would sit waiting on a stdin that never
// answers.
func Interactive() bool {
	if os.Getenv("CI") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// promptWidth is the target inner width for interactive prompts. Forms are pinned
// to this so titles and descriptions land at the same place from one prompt to
// the next, instead of huh sizing each card to its own content. It mirrors the
// error card's 70-column inner width.
const promptWidth = 70

// promptWidthMin keeps prompts usable on very narrow terminals.
const promptWidthMin = 24

// formWidth returns the width to pin a prompt form to: promptWidth, clamped down
// to fit a narrow terminal (leaving a small right margin). Returns promptWidth
// when the terminal size can't be determined (e.g. not a TTY).
func formWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return promptWidth
	}
	avail := w - 2 // small right margin
	if avail >= promptWidth {
		return promptWidth
	}
	if avail < promptWidthMin {
		return promptWidthMin
	}
	return avail
}

// cardWidth returns the inner content width for a bordered card (RenderUserError,
// RenderCrash): the card's natural 70 columns, clamped to fit a narrow terminal
// (the two border cells are outside this width). Falls back to the full width
// when neither stderr nor stdout is a TTY.
func cardWidth() int {
	w, ok := terminalWidth()
	if !ok {
		return promptWidth
	}
	avail := w - 2 // the ╭…╮ / │…│ border cells
	switch {
	case avail >= promptWidth:
		return promptWidth
	case avail < promptWidthMin:
		return promptWidthMin
	default:
		return avail
	}
}

// terminalWidth returns the controlling terminal's column count. It tries stderr
// then stdout (either is the terminal in practice), so it works for cards on
// either stream. ok is false when neither is a TTY.
func terminalWidth() (int, bool) {
	for _, fd := range []int{int(os.Stderr.Fd()), int(os.Stdout.Fd())} {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			return w, true
		}
	}
	return 0, false
}
