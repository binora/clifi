package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("205") // Pink/magenta
	ColorSuccess   = lipgloss.Color("35")  // Green
	ColorWarning   = lipgloss.Color("214") // Gold/yellow
	ColorError     = lipgloss.Color("196") // Red
	ColorDim       = lipgloss.Color("241") // Gray
	ColorAccent    = lipgloss.Color("39")  // Blue
	ColorHighlight = lipgloss.Color("212") // Light pink
)

const (
	SymbolPrompt    = "❯"
	SymbolBullet    = "●"
	SymbolTree      = "└"
	SymbolArrow     = "▸"
	SymbolCheck     = "✓"
	SymbolCross     = "✗"
	SymbolThinking  = "◐"
	SymbolTreeBranch = "├"
	SymbolTreePipe   = "│"
)

var (
	PromptStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	UserStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	AssistantStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	ToolCallStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	SystemStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	SelectorCursor = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	SelectorItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	SelectorDim = lipgloss.NewStyle().
			Foreground(ColorDim)

	SelectorActive = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorDim)
)
