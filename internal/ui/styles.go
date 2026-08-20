package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.Color("39")
	colorMuted  = lipgloss.Color("240")
	colorError  = lipgloss.Color("203")
	colorOk     = lipgloss.Color("42")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	helpStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	errorStyle = lipgloss.NewStyle().Foreground(colorError)
	okStyle    = lipgloss.NewStyle().Foreground(colorOk)
	linkStyle  = lipgloss.NewStyle().Foreground(colorAccent).Underline(true)

	panelStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorMuted).Padding(0, 1)
	panelActiveStyle = panelStyle.BorderForeground(colorAccent)
)

func panelStyleFor(active bool) lipgloss.Style {
	if active {
		return panelActiveStyle
	}
	return panelStyle
}
