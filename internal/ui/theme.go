package ui

import "charm.land/lipgloss/v2"

// Error-card palette (mirrors the driverforge CLI). The border + title carry the
// accent; the message is clean and readable; the tip pops in a brighter tone.
var (
	colAmber = lipgloss.Color("#D9A441") // accent: title + border for user errors
	colTip   = lipgloss.Color("#F4D58D") // brighter gold — the actionable tip
	colInk   = lipgloss.Color("#F2ECE0") // warm off-white — primary message text
	colRed   = lipgloss.Color("#E5534B") // accent: title + border for crashes
	colDim   = lipgloss.Color("#9B948A") // dimmed technical detail
	colBack  = lipgloss.Color("#1C1B19") // dark ink for text on an accent fill
)

// Log-line palette: the chalk colours the Node CLI used, so `gayle` output reads
// the same. lipgloss + the colorprofile writer degrade to plain text on pipes,
// dumb terminals, and NO_COLOR — matching chalk's TTY detection.
var (
	grayStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // chalk.gray — status lines
	whiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))   // chalk.white — info lines
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // chalk.cyan — banners/headers
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // chalk.green — Done. / prompt labels
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // chalk.yellow — warnings, deletions
)
