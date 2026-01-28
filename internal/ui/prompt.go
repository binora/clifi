package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Prompt is a single-line input with a styled prefix
type Prompt struct {
	input   textinput.Model
	width   int
	focused bool
}

// NewPrompt creates a new prompt component
func NewPrompt() Prompt {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 2000
	ti.Width = 80

	return Prompt{
		input:   ti,
		width:   80,
		focused: true,
	}
}

// Focus sets focus on the prompt
func (p *Prompt) Focus() tea.Cmd {
	p.focused = true
	return p.input.Focus()
}

// Blur removes focus from the prompt
func (p *Prompt) Blur() {
	p.focused = false
	p.input.Blur()
}

// Focused returns whether the prompt has focus
func (p *Prompt) Focused() bool {
	return p.focused
}

// SetWidth sets the width of the input
func (p *Prompt) SetWidth(w int) {
	p.width = w
	p.input.Width = w - 4 // Account for prompt symbol and spacing
}

// Value returns the current input value
func (p *Prompt) Value() string {
	return p.input.Value()
}

// SetValue sets the input value
func (p *Prompt) SetValue(s string) {
	p.input.SetValue(s)
}

// Reset clears the input
func (p *Prompt) Reset() {
	p.input.Reset()
}

// Update handles input events
func (p *Prompt) Update(msg tea.Msg) (*Prompt, tea.Cmd) {
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

// View renders the prompt
func (p *Prompt) View() string {
	style := SelectorDim
	if p.focused {
		style = PromptStyle
	}
	return style.Render(SymbolPrompt) + " " + p.input.View()
}
