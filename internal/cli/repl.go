package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yolodolo42/clifi/internal/agent"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("35"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// chatMessage represents a message in the chat history
type chatMessage struct {
	role    string // "user", "assistant", "error", "system"
	content string
	time    time.Time
}

// model represents the REPL state
type model struct {
	agent    *agent.Agent
	textarea textarea.Model
	viewport viewport.Model
	messages []chatMessage
	spinner  spinner.Model
	loading  bool
	width    int
	height   int
	ready    bool
	quitting bool
}

// responseMsg is sent when the agent responds
type responseMsg struct {
	content string
	err     error
}

// initialModel creates the initial model state
func initialModel(ag *agent.Agent) model {
	ta := textarea.New()
	ta.Placeholder = "Ask me anything about your crypto..."
	ta.Focus()
	ta.CharLimit = 500
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		agent:    ag,
		textarea: ta,
		spinner:  sp,
		messages: []chatMessage{
			{
				role:    "system",
				content: "Welcome to clifi! I'm your crypto operator agent.\nType your questions below. Use /help for commands, /quit to exit.",
				time:    time.Now(),
			},
		},
	}
}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

// Update handles messages and updates state
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

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

			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			// Handle commands
			if strings.HasPrefix(input, "/") {
				m.textarea.Reset()
				return m.handleCommand(input)
			}

			// Add user message
			m.messages = append(m.messages, chatMessage{
				role:    "user",
				content: input,
				time:    time.Now(),
			})

			// Clear input and start loading
			m.textarea.Reset()
			m.loading = true
			m.updateViewport()

			// Send to agent
			return m, m.sendToAgent(input)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-8)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 8
		}
		m.textarea.SetWidth(msg.Width - 4)
		m.updateViewport()

	case responseMsg:
		m.loading = false
		if msg.err != nil {
			m.messages = append(m.messages, chatMessage{
				role:    "error",
				content: msg.err.Error(),
				time:    time.Now(),
			})
		} else {
			m.messages = append(m.messages, chatMessage{
				role:    "assistant",
				content: msg.content,
				time:    time.Now(),
			})
		}
		m.updateViewport()
		m.viewport.GotoBottom()

	case spinner.TickMsg:
		m.spinner, spCmd = m.spinner.Update(msg)
		return m, spCmd
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
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

	// Title
	title := titleStyle.Render("  clifi - Crypto Operator Agent")
	b.WriteString(title + "\n\n")

	// Messages viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Loading indicator or input
	if m.loading {
		b.WriteString(fmt.Sprintf("\n  %s Thinking...\n\n", m.spinner.View()))
	} else {
		b.WriteString("\n")
	}

	// Input area
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	// Help
	help := helpStyle.Render("  /help • /model • /clear • /quit • Ctrl+C to exit")
	b.WriteString(help)

	return b.String()
}

// updateViewport updates the viewport content with messages
func (m *model) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			content.WriteString(userStyle.Render("You: "))
			content.WriteString(msg.content)
		case "assistant":
			content.WriteString(assistantStyle.Render("clifi: "))
			content.WriteString(msg.content)
		case "error":
			content.WriteString(errorStyle.Render("Error: "))
			content.WriteString(msg.content)
		case "system":
			content.WriteString(helpStyle.Render(msg.content))
		}
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
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
				role:    "system",
				content: "Chat cleared. How can I help you?",
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
		helpText := `Available commands:
  /help, /?       - Show this help
  /model          - List available models
  /model <id>     - Switch to a different model
  /clear          - Clear chat history
  /quit, /exit    - Exit clifi

Example queries:
  "What's my portfolio?"
  "Show my ETH balance on Base"
  "What chains are supported?"
  "List my wallets"`

		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: helpText,
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil

	default:
		m.messages = append(m.messages, chatMessage{
			role:    "error",
			content: fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd),
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}
}

// handleModelCommand lists models or switches to a new one
func (m model) handleModelCommand(modelID string) (tea.Model, tea.Cmd) {
	if m.agent == nil {
		m.messages = append(m.messages, chatMessage{
			role:    "error",
			content: "Agent not initialized.",
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	// No argument: list available models
	if modelID == "" {
		current := m.agent.CurrentModel()
		models := m.agent.ListModels()
		provider := m.agent.ProviderName()

		var b strings.Builder
		b.WriteString(fmt.Sprintf("Models for %s:\n", provider))
		for _, md := range models {
			marker := "  "
			if md.ID == current {
				marker = "▸ "
			}
			toolTag := ""
			if !md.SupportsTools {
				toolTag = " (no tool support)"
			}
			b.WriteString(fmt.Sprintf("  %s%-30s %s%s\n", marker, md.ID, md.Name, toolTag))
		}
		b.WriteString(fmt.Sprintf("\nActive: %s", current))
		b.WriteString("\nUsage: /model <id>")

		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: b.String(),
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	// Switch model
	if err := m.agent.SetModel(modelID); err != nil {
		m.messages = append(m.messages, chatMessage{
			role:    "error",
			content: fmt.Sprintf("Failed to switch model: %v", err),
			time:    time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	m.messages = append(m.messages, chatMessage{
		role:    "system",
		content: fmt.Sprintf("Switched to %s. Conversation history cleared.", modelID),
		time:    time.Now(),
	})
	m.updateViewport()
	return m, nil
}

// sendToAgent sends a message to the agent and returns a command
func (m model) sendToAgent(input string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		response, err := m.agent.Chat(ctx, input)
		return responseMsg{
			content: response,
			err:     err,
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
