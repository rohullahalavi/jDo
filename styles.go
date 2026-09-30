// styles.go — lipgloss styles for the plain CLI output only.
// The TUI styles live inside tui_view.go so the TUI is self contained.
package main

import "github.com/charmbracelet/lipgloss"

var (
	pHead   = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
	pRule   = lipgloss.NewStyle().Foreground(colFaint)
	pNum    = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
	pName   = lipgloss.NewStyle().Foreground(colWhite)
	pText   = lipgloss.NewStyle().Foreground(colWhite)
	pDim    = lipgloss.NewStyle().Foreground(colDim)
	pFaint  = lipgloss.NewStyle().Foreground(colFaint)
	pPrompt = lipgloss.NewStyle().Foreground(colBlue)
	pFail   = lipgloss.NewStyle().Foreground(colWhite).Bold(true)
	pOK     = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
	pCmd    = lipgloss.NewStyle().Foreground(colBlue)
	pBlue   = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
)
