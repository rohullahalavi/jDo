// meta.go — application metadata and the strict three-hue palette.
// The entire app (CLI and TUI) uses only black, white and blue.
// dim and faint are intensity steps of white, not separate hues.
package main

import "github.com/charmbracelet/lipgloss"

const (
	appName    = "jdo"
	appVersion = "1.5.0"
	appTagline = "markdown notes commander"
)

var (
	colBlack = lipgloss.Color("#000000") // every background
	colWhite = lipgloss.Color("#ffffff") // primary text
	colBlue  = lipgloss.Color("#4c9aff") // accent: badges, indicators, keys
	colSelBg = lipgloss.Color("#2f6fed") // selected row background (solid blue)
	colDim   = lipgloss.Color("#9a9a9a") // secondary text (dim white)
	colFaint = lipgloss.Color("#3a3a3a") // borders and dividers (faint white)
)
