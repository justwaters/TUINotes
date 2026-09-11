package ui

import "charm.land/lipgloss/v2"

var (
	colorAccent   = lipgloss.Color("62")
	colorMuted    = lipgloss.Color("240")
	colorErr      = lipgloss.Color("204")
	colorBorder   = lipgloss.Color("238")
	colorBorderOn = lipgloss.Color("62")

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	paneStyleFocused = paneStyle.BorderForeground(colorBorderOn)

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	statusStyle = lipgloss.NewStyle().Foreground(colorMuted)

	errStyle = lipgloss.NewStyle().Foreground(colorErr).Bold(true)
)
