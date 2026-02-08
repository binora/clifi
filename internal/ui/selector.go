package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SelectorItem represents an item in the selector
type SelectorItem struct {
	ID          string
	Label       string
	Description string
	Current     bool
}

// Selector is an interactive list selector
type Selector struct {
	title    string
	items    []SelectorItem
	cursor   int
	selected int
	active   bool
	width    int
}

// NewSelector creates a new selector
func NewSelector(title string, items []SelectorItem) Selector {
	// Find currently selected item
	selected := 0
	for i, item := range items {
		if item.Current {
			selected = i
			break
		}
	}

	return Selector{
		title:    title,
		items:    items,
		cursor:   selected,
		selected: selected,
		active:   true,
		width:    80,
	}
}

// SetWidth sets the selector width
func (s *Selector) SetWidth(w int) {
	s.width = w
}

// Active returns whether the selector is active
func (s *Selector) Active() bool {
	return s.active
}

// Selected returns the selected item ID, or empty if cancelled
func (s *Selector) Selected() string {
	if s.selected >= 0 && s.selected < len(s.items) {
		return s.items[s.selected].ID
	}
	return ""
}

// Cancelled returns whether the selector was cancelled
func (s *Selector) Cancelled() bool {
	return !s.active && s.selected == -1
}

// Update handles selector input
func (s *Selector) Update(msg tea.Msg) (*Selector, tea.Cmd) {
	if !s.active {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}
		case "enter":
			s.selected = s.cursor
			s.active = false
		case "esc", "q":
			s.selected = -1
			s.active = false
		}
	}

	return s, nil
}

// View renders the selector
func (s *Selector) View() string {
	if !s.active {
		return ""
	}

	var b strings.Builder

	b.WriteString(HelpStyle.Render(s.title + " (↑/↓ navigate, enter select, esc cancel)"))
	b.WriteString("\n\n")

	for i, item := range s.items {
		isCursor := i == s.cursor

		if isCursor {
			b.WriteString(SelectorCursor.Render(SymbolArrow) + " ")
		} else {
			b.WriteString("  ")
		}

		display := item.Label
		if display == "" {
			display = item.ID
		}
		label := fmt.Sprintf("%-35s", display)
		if isCursor {
			b.WriteString(SelectorActive.Render(label))
		} else {
			b.WriteString(SelectorItemStyle.Render(label))
		}

		if item.Description != "" {
			desc := item.Description
			if item.Current {
				desc += " (current)"
			}
			b.WriteString(SelectorDim.Render(desc))
		}

		b.WriteString("\n")
	}

	return b.String()
}
