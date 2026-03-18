// Package ui implements the Bubble Tea TUI for jala.
package ui

import "github.com/charmbracelet/lipgloss"

// Colour palette — designed to work on both dark and light terminals.
// Uses adaptive colours so it degrades gracefully on 256-colour/no-colour terms.
var (
	colorPrimary  = lipgloss.AdaptiveColor{Light: "#5A4FCF", Dark: "#9D8FFF"}
	colorAccent   = lipgloss.AdaptiveColor{Light: "#D64F00", Dark: "#FF8C57"}
	colorMuted    = lipgloss.AdaptiveColor{Light: "#7A7A7A", Dark: "#6C6C6C"}
	colorSuccess  = lipgloss.AdaptiveColor{Light: "#1A7A1A", Dark: "#50FA7B"}
	colorDanger   = lipgloss.AdaptiveColor{Light: "#C00000", Dark: "#FF5555"}
	colorBorder   = lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#3A3A3A"}
	colorSelf     = lipgloss.AdaptiveColor{Light: "#0055CC", Dark: "#8BE9FD"}
	colorSystem   = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}

	// App chrome
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1E1E1E"}).
			Foreground(colorMuted).
			Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Inherit(styleStatusBar).
			Foreground(colorPrimary).
			Bold(true)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	// Message lines
	styleMsgTime = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleMsgNickSelf = lipgloss.NewStyle().
				Foreground(colorSelf).
				Bold(true)

	styleMsgNickOther = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	styleMsgBody = lipgloss.NewStyle()

	styleMsgSystem = lipgloss.NewStyle().
			Foreground(colorSystem).
			Italic(true)

	// Input area
	styleInput = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, false).
			Padding(0, 1)

	styleInputPrompt = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// Peer list
	stylePeerList = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	stylePeerItem = lipgloss.NewStyle().
			Foreground(colorSuccess)

	stylePeerCount = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// Help bar
	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colorPrimary)
)
