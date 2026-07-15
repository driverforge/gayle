package ui

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/driverforge/gayle/internal/clierr"
)

// RenderUserError writes an expected, user-facing error to w. On a terminal it
// renders a titled card: a short label in the top border, the message, and —
// grouped beneath it — a brighter "💡" tip. In a pipe/CI it writes the Node
// CLI's grep-able "Gayle: ERROR: <msg>" line (plus a "Gayle: TIP:" line when
// there is a hint).
func RenderUserError(w io.Writer, err error) {
	if err == nil {
		return
	}
	title, msg, hint := dissect(err)
	if msg == "" && hint == "" {
		return // silent error (clierr.Silent): exit non-zero, render nothing
	}

	if !isTerminalWriter(w) {
		fmt.Fprintln(w, "Gayle: ERROR: "+msg)
		if hint != "" {
			fmt.Fprintln(w, "Gayle: TIP: "+hint)
		}
		return
	}

	rows := []string{textRow(colInk, true, msg)}
	if hint != "" {
		rows = append(rows, "", tipRow("💡  "+linkify(hint)))
	}
	fmt.Fprintln(w) // breathing room from the command line above
	Fprintln(w, card(colAmber, title, rows))
}

// RenderCrash writes an unexpected error (a bug) to w. On a terminal it renders
// a red titled card with an apologetic line and the underlying error as dimmed
// technical detail; in a pipe/CI it writes the plain "Gayle: ERROR:" line.
func RenderCrash(w io.Writer, err error) {
	if err == nil {
		return
	}
	if !isTerminalWriter(w) {
		fmt.Fprintln(w, "Gayle: ERROR: "+err.Error())
		return
	}

	rows := []string{
		textRow(colInk, true, "Something went wrong on our end — sorry about that."),
		"",
		textRow(colDim, false, err.Error()),
	}
	fmt.Fprintln(w) // breathing room from the command line above
	Fprintln(w, card(colRed, "Error", rows))
}

// card frames pre-rendered content rows in a rounded box whose top border
// carries an uppercase title tab: ╭─ TITLE ─────────╮. Empty rows become spacer
// rows (top and bottom breathing room are added automatically). The inner width
// is measured from a rendered spacer so the borders always line up with the
// body, regardless of padding/emoji width quirks.
func card(accent color.Color, title string, rows []string) string {
	bar := lipgloss.NewStyle().Foreground(accent)
	label := lipgloss.NewStyle().Bold(true).Foreground(colBack).Background(accent)

	spacer := textRow(colInk, false, "")
	width := lipgloss.Width(spacer)

	tab := label.Render(" " + strings.ToUpper(title) + " ")
	dashes := width - 1 - lipgloss.Width(tab)
	if dashes < 0 {
		dashes = 0
	}
	top := bar.Render("╭─") + tab + bar.Render(strings.Repeat("─", dashes)+"╮")
	bottom := bar.Render("╰" + strings.Repeat("─", width) + "╯")

	lines := []string{spacer} // top breathing room
	for _, row := range rows {
		if row == "" {
			lines = append(lines, spacer)
			continue
		}
		lines = append(lines, strings.Split(row, "\n")...)
	}
	lines = append(lines, spacer) // bottom breathing room

	var b strings.Builder
	b.WriteString(top + "\n")
	for _, line := range lines {
		b.WriteString(bar.Render("│") + line + bar.Render("│") + "\n")
	}
	b.WriteString(bottom)
	return b.String()
}

// textRow renders s into a card-width row (wrapping if needed), in the given
// colour, optionally bold.
func textRow(fg color.Color, bold bool, s string) string {
	return lipgloss.NewStyle().Foreground(fg).Bold(bold).Width(cardWidth()).Padding(0, 2).Render(s)
}

// tipRow renders the actionable tip line in the brighter accent tone.
func tipRow(s string) string {
	return textRow(colTip, true, s)
}

var urlRe = regexp.MustCompile(`https?://[^\s]+`)

// linkify wraps any URL in s as an OSC-8 hyperlink, so terminals that support it
// make the link clickable. The visible text is unchanged (others just see the
// URL), and the escapes are zero-width, so card layout is unaffected.
func linkify(s string) string {
	return urlRe.ReplaceAllStringFunc(s, func(u string) string {
		return termenv.Hyperlink(u, u)
	})
}

// dissect pulls a title, message and hint out of err. A clierr.UserError
// supplies them directly (defaulting the title to "Heads up"); otherwise the
// message is the error text as-is.
func dissect(err error) (title, msg, hint string) {
	var ue *clierr.UserError
	if errors.As(err, &ue) {
		title = ue.Title
		if title == "" {
			title = "Heads up"
		}
		return title, ue.Message, ue.Hint
	}
	return "Heads up", err.Error(), ""
}

// isTerminalWriter reports whether w is a terminal (so we should colourise).
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
