package cli

import (
	"github.com/charmbracelet/lipgloss"
)

// Symbols used as line prefixes by the Reporter. The contract limits visual
// glyphs to this set — see .claude/rules/output-contract.md.
//
// symPending and symSkip are reserved for the multi-item Checklist primitive
// (pending row and skipped row respectively); the rest are emitted by their
// corresponding Reporter kinds.
const (
	symStep    = "⏳"
	symSuccess = "✓"
	symInfo    = "ℹ"
	symWarn    = "⚠"
	symError   = "✗"
	symPending = "○"
	symSkip    = "→"
)

// styles holds the lipgloss styles used by the Reporter. Centralizing them
// here is the only legal place to define ANSI/color in the codebase.
var styles = struct {
	step    lipgloss.Style
	success lipgloss.Style
	info    lipgloss.Style
	warn    lipgloss.Style
	errorS  lipgloss.Style

	header    lipgloss.Style
	headerBar lipgloss.Style
	subtle    lipgloss.Style

	box      lipgloss.Style
	boxTitle lipgloss.Style

	tableHeader lipgloss.Style
	tableBorder lipgloss.Color

	diffAdd   lipgloss.Style
	diffDel   lipgloss.Style
	diffHunk  lipgloss.Style
	diffMeta  lipgloss.Style
	diffPlain lipgloss.Style
}{
	step:    lipgloss.NewStyle().Foreground(lipgloss.Color("39")),  // cyan-ish
	success: lipgloss.NewStyle().Foreground(lipgloss.Color("42")),  // green
	info:    lipgloss.NewStyle().Foreground(lipgloss.Color("33")),  // blue
	warn:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")), // amber
	errorS:  lipgloss.NewStyle().Foreground(lipgloss.Color("203")), // red

	header: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("231")).
		PaddingLeft(1).
		PaddingRight(1),
	headerBar: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	subtle:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),

	box: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1),
	boxTitle: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("231")),

	tableHeader: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("231")),
	tableBorder: lipgloss.Color("240"),

	diffAdd:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	diffDel:   lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
	diffHunk:  lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
	diffMeta:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	diffPlain: lipgloss.NewStyle(),
}

// HeadingText renders a label with bold + bright color, used for primary
// names (region values, application names) in the lsresources tree view.
func HeadingText(s string) string {
	return styles.boxTitle.Render(s)
}

// SubtleText renders a string in a dim color, used for secondary metadata
// (resource IDs, parenthetical hints) in the lsresources tree view.
func SubtleText(s string) string {
	return styles.subtle.Render(s)
}

// StateBadge renders an AppConfig deployment state with a state-appropriate
// color. lipgloss honors NO_COLOR and strips ANSI when rendering to a
// non-terminal, so callers can hand the result straight to Reporter.Table
// without affecting layout in piped/CI output.
func StateBadge(state string) string {
	switch state {
	case "COMPLETE":
		return styles.success.Render(state)
	case "DEPLOYING", "BAKING":
		return styles.warn.Render(state)
	case "ROLLED_BACK", "ROLLING_BACK":
		return styles.errorS.Render(state)
	default:
		return state
	}
}
