package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yolodolo42/clifi/internal/agent"
	"github.com/yolodolo42/clifi/internal/ui"
)

// replMode represents the current interaction mode
type replMode int

const (
	modeChat replMode = iota
	modeModelSelector
)

// chatMessage represents a message in the chat history
type chatMessage struct {
	kind     string // "user", "tool_call", "tool_result", "assistant", "error", "system"
	content  string
	toolName string
	toolArgs string
	time     time.Time
}

// model represents the REPL state
type model struct {
	agent         *agent.Agent
	prompt        ui.Prompt
	viewport      viewport.Model
	messages      []chatMessage
	spinner       spinner.Model
	loading       bool
	width         int
	height        int
	ready         bool
	quitting      bool
	mode          replMode
	modelSelector ui.Selector
}

// responseMsg is sent when the agent responds
type responseMsg struct {
	events []agent.ChatEvent
	err    error
}

// initialModel creates the initial model state
func initialModel(ag *agent.Agent) model {
	prompt := ui.NewPrompt()
	prompt.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ui.ColorWarning)

	return model{
		agent:   ag,
		prompt:  prompt,
		spinner: sp,
		mode:    modeChat,
		messages: []chatMessage{
			{
				kind:    "system",
				content: "Welcome to clifi! Type your questions below. Use /help for commands.",
				time:    time.Now(),
			},
		},
	}
}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return tea.Batch(m.prompt.Focus(), m.spinner.Tick)
}

// Update handles messages and updates state
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle mode-specific updates
	switch m.mode {
	case modeModelSelector:
		return m.updateModelSelector(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}

			input := strings.TrimSpace(m.prompt.Value())
			if input == "" {
				return m, nil
			}

			// Handle commands
			if strings.HasPrefix(input, "/") {
				m.prompt.Reset()
				return m.handleCommand(input)
			}

			// Add user message
			m.messages = append(m.messages, chatMessage{
				kind:    "user",
				content: input,
				time:    time.Now(),
			})

			// Clear input and start loading
			m.prompt.Reset()
			m.loading = true
			m.updateViewport()

			// Send to agent
			return m, m.sendToAgent(input)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}
		m.prompt.SetWidth(msg.Width - 2)
		m.updateViewport()

	case responseMsg:
		m.loading = false
		if msg.err != nil {
			m.messages = append(m.messages, chatMessage{
				kind:    "error",
				content: msg.err.Error(),
				time:    time.Now(),
			})
		} else {
			// Add events as messages
			for _, event := range msg.events {
				switch event.Type {
				case "tool_call":
					m.messages = append(m.messages, chatMessage{
						kind:     "tool_call",
						toolName: event.Tool,
						toolArgs: event.Args,
						time:     time.Now(),
					})
				case "tool_result":
					m.messages = append(m.messages, chatMessage{
						kind:     "tool_result",
						toolName: event.Tool,
						content:  event.Content,
						time:     time.Now(),
					})
				case "content":
					m.messages = append(m.messages, chatMessage{
						kind:    "assistant",
						content: event.Content,
						time:    time.Now(),
					})
				}
			}
		}
		m.updateViewport()
		m.viewport.GotoBottom()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update prompt
	var promptCmd tea.Cmd
	promptPtr, promptCmd := m.prompt.Update(msg)
	m.prompt = *promptPtr
	cmds = append(cmds, promptCmd)

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// updateModelSelector handles input in model selector mode
func (m model) updateModelSelector(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		selectorPtr, _ := m.modelSelector.Update(msg)
		m.modelSelector = *selectorPtr

		if !m.modelSelector.Active() {
			m.mode = modeChat
			if !m.modelSelector.Cancelled() {
				selected := m.modelSelector.Selected()
				if selected != "" && selected != m.agent.CurrentModel() {
					if err := m.agent.SetModel(selected); err != nil {
						m.messages = append(m.messages, chatMessage{
							kind:    "error",
							content: fmt.Sprintf("Failed to switch model: %v", err),
							time:    time.Now(),
						})
					} else {
						m.messages = append(m.messages, chatMessage{
							kind:    "system",
							content: fmt.Sprintf("Switched to %s. Conversation cleared.", selected),
							time:    time.Now(),
						})
					}
				}
			}
			m.updateViewport()
			return m, m.prompt.Focus()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.modelSelector.SetWidth(msg.Width)
	}

	return m, nil
}

// View renders the UI
func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Initializing...\n"
	}

	var b strings.Builder

	// Model selector mode
	if m.mode == modeModelSelector {
		b.WriteString("\n")
		b.WriteString(m.modelSelector.View())
		return b.String()
	}

	// Chat mode
	// Messages viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Loading indicator
	if m.loading {
		b.WriteString(fmt.Sprintf("  %s Thinking...\n", m.spinner.View()))
	}

	// Input prompt
	b.WriteString(m.prompt.View())
	b.WriteString("\n")

	// Help footer
	help := ui.HelpStyle.Render("  /help • /model • /clear • /quit")
	b.WriteString(help)

	return b.String()
}

// updateViewport updates the viewport content with messages
func (m *model) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.kind {
		case "user":
			content.WriteString(ui.PromptStyle.Render(ui.SymbolPrompt))
			content.WriteString(" ")
			content.WriteString(msg.content)

		case "tool_call":
			content.WriteString(ui.ToolCallStyle.Render(ui.SymbolBullet))
			content.WriteString(" ")
			content.WriteString(ui.ToolCallStyle.Render(msg.toolName))
			content.WriteString(ui.SelectorDim.Render("("))
			args := summarizeArgs(msg.toolArgs, m.width-len(msg.toolName)-10)
			content.WriteString(ui.SelectorDim.Render(args))
			content.WriteString(ui.SelectorDim.Render(")"))

		case "tool_result":
			lines := strings.Split(msg.content, "\n")
			for i, line := range lines {
				if i == 0 {
					content.WriteString("  ")
					content.WriteString(ui.ToolResultStyle.Render(ui.SymbolTree))
					content.WriteString(" ")
				} else {
					content.WriteString("    ")
				}
				content.WriteString(ui.ToolResultStyle.Render(line))
				if i < len(lines)-1 {
					content.WriteString("\n")
				}
			}

		case "assistant":
			content.WriteString(ui.AssistantStyle.Render(ui.SymbolBullet))
			content.WriteString(" ")
			content.WriteString(msg.content)

		case "error":
			content.WriteString(ui.ErrorStyle.Render(ui.SymbolBullet))
			content.WriteString(" ")
			content.WriteString(ui.ErrorStyle.Render("Error: "))
			content.WriteString(msg.content)

		case "system":
			content.WriteString(ui.SystemStyle.Render(msg.content))
		}
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
}

// summarizeArgs truncates tool args for display
func summarizeArgs(args string, maxLen int) string {
	if maxLen < 20 {
		maxLen = 20
	}
	// Remove newlines and extra whitespace
	args = strings.ReplaceAll(args, "\n", " ")
	args = strings.Join(strings.Fields(args), " ")

	if len(args) <= maxLen {
		return args
	}
	return args[:maxLen-3] + "..."
}

// handleCommand handles slash commands
func (m model) handleCommand(input string) (tea.Model, tea.Cmd) {
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "/quit", "/exit", "/q":
		m.quitting = true
		return m, tea.Quit

	case "/clear":
		m.messages = []chatMessage{
			{
				kind:    "system",
				content: "Chat cleared.",
				time:    time.Now(),
			},
		}
		if m.agent != nil {
			m.agent.Reset()
		}
		m.updateViewport()
		return m, nil

	case "/model":
		return m.handleModelCommand(arg)

	case "/help", "/?":
		helpText := `Commands:
  /help       Show this help
  /model      Select a model interactively
  /model <id> Switch to a specific model
  /clear      Clear chat history
  /quit       Exit clifi`

		m.messages = append(m.messages, chatMessage{
			kind:    "system",
			content: helpText,
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil

	default:
		m.messages = append(m.messages, chatMessage{
			kind:    "error",
			content: fmt.Sprintf("Unknown command: %s. Type /help for commands.", cmd),
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}
}

// handleModelCommand shows model selector or switches directly
func (m model) handleModelCommand(modelID string) (tea.Model, tea.Cmd) {
	if m.agent == nil {
		m.messages = append(m.messages, chatMessage{
			kind:    "error",
			content: "Agent not initialized.",
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	// Direct switch if model ID provided
	if modelID != "" {
		if err := m.agent.SetModel(modelID); err != nil {
			m.messages = append(m.messages, chatMessage{
				kind:    "error",
				content: fmt.Sprintf("Failed to switch model: %v", err),
				time:    time.Now(),
			})
			m.updateViewport()
			return m, nil
		}

		m.messages = append(m.messages, chatMessage{
			kind:    "system",
			content: fmt.Sprintf("Switched to %s. Conversation cleared.", modelID),
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	// Interactive selector
	current := m.agent.CurrentModel()
	models := m.agent.ListModels()
	provider := m.agent.ProviderName()

	items := make([]ui.SelectorItem, len(models))
	for i, md := range models {
		items[i] = ui.SelectorItem{
			ID:          md.ID,
			Label:       md.ID,
			Description: md.Name,
			Current:     md.ID == current,
		}
	}

	m.modelSelector = ui.NewSelector(fmt.Sprintf("Select %s model", provider), items)
	m.modelSelector.SetWidth(m.width)
	m.mode = modeModelSelector
	m.prompt.Blur()

	return m, nil
}

// sendToAgent sends a message to the agent and returns a command
func (m model) sendToAgent(input string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		events, err := m.agent.ChatWithEvents(ctx, input)
		return responseMsg{
			events: events,
			err:    err,
		}
	}
}

// RunREPL starts the interactive REPL
func RunREPL() error {
	ag, err := agent.New("")
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	defer ag.Close()

	p := tea.NewProgram(
		initialModel(ag),
		tea.WithAltScreen(),
	)

	_, err = p.Run()
	return err
}
