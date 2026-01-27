package setup

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("205") // Pink/magenta
	successColor   = lipgloss.Color("35")  // Green
	dimColor       = lipgloss.Color("241") // Gray
	accentColor    = lipgloss.Color("39")  // Blue
	borderColor    = lipgloss.Color("62")  // Purple
	highlightColor = lipgloss.Color("212") // Light pink

	// Box style for welcome/complete screens
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	// Subtitle/description
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Dim text
	DimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Step indicator (e.g., "Step 1 of 2")
	StepStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Selected item in list
	SelectedStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true)

	// Normal item in list
	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Cursor
	CursorStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// Error text
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // Red

	// Help text at bottom
	HelpStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Checkmark
	Checkmark = SuccessStyle.Render("✓")

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)
)
