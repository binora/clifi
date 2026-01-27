package setup

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yolodolo42/clifi/internal/auth"
	"github.com/yolodolo42/clifi/internal/llm"
	"golang.org/x/term"
)

// WizardStep represents the current step in the wizard
type WizardStep int

const (
	StepWelcome WizardStep = iota
	StepProviderSelect
	StepAuthMethod
	StepProviderKey
	StepOAuthWaiting
	StepWalletChoice
	StepWalletPassword
	StepWalletConfirm
	StepComplete
)

const totalSteps = 3 // Provider, Wallet, Complete

// SetupResult contains the result of the setup wizard
type SetupResult struct {
	ProviderID    llm.ProviderID
	WalletCreated bool
	WalletAddress string
	Cancelled     bool
}

// WizardModel is the main wizard Bubbletea model
type WizardModel struct {
	step     WizardStep
	status   *SetupStatus
	dataDir  string
	width    int
	height   int
	err      error
	quitting bool

	// Provider step
	providerList     []providerItem
	providerCursor   int
	selectedProvider llm.ProviderID
	apiKeyInput      textinput.Model
	validatingKey    bool
	keyValidated     bool
	keyError         string
	envKeyDetected   bool
	envKeyProvider   llm.ProviderID

	// Auth method step
	authMethods      []authMethodItem
	authMethodCursor int
	selectedAuth     string // "api" or "oauth"
	oauthInProgress  bool
	oauthError       string

	// Wallet step
	walletChoices []string
	walletCursor  int
	walletChoice  string
	passwordInput textinput.Model
	confirmInput  textinput.Model
	passwordStep  int // 0=enter, 1=confirm
	passwordError string
	walletCreated bool
	walletAddress string

	// UI
	spinner  spinner.Model
	progress progress.Model

	// Result
	result *SetupResult
}

type providerItem struct {
	id          llm.ProviderID
	name        string
	description string
	recommended bool
}

type authMethodItem struct {
	authType    string // "api" or "oauth"
	label       string
	description string
}

// Message types
type keyValidatedMsg struct {
	success bool
	err     error
}

type oauthCompleteMsg struct {
	success bool
	err     error
}

type walletCreatedMsg struct {
	address string
	err     error
}

// NewWizard creates a new wizard model
func NewWizard(dataDir string) *WizardModel {
	status, _ := DetectSetupStatus(dataDir)

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	// Progress bar
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	// API key input
	apiInput := textinput.New()
	apiInput.Placeholder = "Paste your API key here..."
	apiInput.EchoMode = textinput.EchoPassword
	apiInput.EchoCharacter = '•'
	apiInput.CharLimit = 200
	apiInput.Width = 50

	// Password inputs
	passInput := textinput.New()
	passInput.Placeholder = "Enter password (8+ chars)"
	passInput.EchoMode = textinput.EchoPassword
	passInput.EchoCharacter = '•'
	passInput.CharLimit = 100
	passInput.Width = 40

	confirmInput := textinput.New()
	confirmInput.Placeholder = "Confirm password"
	confirmInput.EchoMode = textinput.EchoPassword
	confirmInput.EchoCharacter = '•'
	confirmInput.CharLimit = 100
	confirmInput.Width = 40

	providers := []providerItem{
		{id: llm.ProviderAnthropic, name: "Anthropic (Claude)", description: "Best reasoning & tool use", recommended: true},
		{id: llm.ProviderOpenAI, name: "OpenAI (GPT-4)", description: "Fast responses, widely used"},
		{id: llm.ProviderGemini, name: "Google (Gemini)", description: "1M token context window"},
		{id: llm.ProviderCopilot, name: "GitHub Copilot", description: "Free with Copilot subscription"},
		{id: llm.ProviderVenice, name: "Venice AI", description: "Privacy-focused, uncensored"},
		{id: llm.ProviderOpenRouter, name: "OpenRouter", description: "Access 100+ models with one key"},
	}

	walletChoices := []string{
		"Create a new wallet",
		"Import existing wallet (coming soon)",
		"Continue without wallet",
	}

	m := &WizardModel{
		step:          StepWelcome,
		status:        status,
		dataDir:       dataDir,
		providerList:  providers,
		walletChoices: walletChoices,
		spinner:       sp,
		progress:      prog,
		apiKeyInput:   apiInput,
		passwordInput: passInput,
		confirmInput:  confirmInput,
	}

	// Check for environment keys
	m.detectEnvKeys()

	// Skip provider step if already configured
	if status.HasProvider {
		m.selectedProvider = status.ProviderID
		m.step = StepWalletChoice
		// Skip wallet step too if already configured
		if status.HasWallet {
			m.walletAddress = status.WalletAddress
			m.step = StepComplete
		}
	}

	return m
}

// detectEnvKeys checks for API keys in environment variables
func (m *WizardModel) detectEnvKeys() {
	for _, p := range m.providerList {
		envVar := llm.EnvVarForProvider(p.id)
		if envVar != "" && os.Getenv(envVar) != "" {
			m.envKeyDetected = true
			m.envKeyProvider = p.id
			return
		}
	}
}

// Init initializes the wizard
func (m WizardModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

// Update handles messages
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.Type {
		case tea.KeyCtrlC:
			m.result = &SetupResult{Cancelled: true}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.step > StepWelcome && m.step < StepComplete {
				m.step--
				m.err = nil
				m.keyError = ""
				m.passwordError = ""
				return m, nil
			}
		}

		// Step-specific handling
		switch m.step {
		case StepWelcome:
			if msg.Type == tea.KeyEnter {
				if m.envKeyDetected {
					// Use detected env key
					m.selectedProvider = m.envKeyProvider
					m.step = StepWalletChoice
				} else {
					m.step = StepProviderSelect
				}
			}
			return m, nil

		case StepProviderSelect:
			return m.updateProviderSelect(msg)

		case StepAuthMethod:
			return m.updateAuthMethod(msg)

		case StepProviderKey:
			return m.updateProviderKey(msg)

		case StepOAuthWaiting:
			// OAuth is in progress, just wait
			return m, nil

		case StepWalletChoice:
			return m.updateWalletChoice(msg)

		case StepWalletPassword:
			return m.updateWalletPassword(msg)

		case StepComplete:
			if msg.Type == tea.KeyEnter {
				m.result = &SetupResult{
					ProviderID:    m.selectedProvider,
					WalletCreated: m.walletCreated,
					WalletAddress: m.walletAddress,
				}
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(40, msg.Width-20)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case keyValidatedMsg:
		m.validatingKey = false
		if msg.success {
			m.keyValidated = true
			m.keyError = ""
			if err := m.saveProviderKey(); err != nil {
				m.keyError = fmt.Sprintf("Failed to save: %v", err)
			} else {
				m.step = StepWalletChoice
			}
		} else {
			m.keyError = formatKeyError(msg.err, m.selectedProvider)
		}
		return m, nil

	case walletCreatedMsg:
		if msg.err != nil {
			m.passwordError = msg.err.Error()
		} else {
			m.walletCreated = true
			m.walletAddress = msg.address
			m.step = StepComplete
		}
		return m, nil

	case oauthCompleteMsg:
		m.oauthInProgress = false
		if msg.success {
			m.step = StepWalletChoice
		} else {
			m.oauthError = msg.err.Error()
			m.step = StepAuthMethod // Go back to auth method selection
		}
		return m, nil
	}

	// Update text inputs
	if m.step == StepProviderKey && !m.validatingKey {
		var cmd tea.Cmd
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.step == StepWalletPassword {
		var cmd tea.Cmd
		if m.passwordStep == 0 {
			m.passwordInput, cmd = m.passwordInput.Update(msg)
		} else {
			m.confirmInput, cmd = m.confirmInput.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// formatKeyError returns a user-friendly error message
func formatKeyError(err error, provider llm.ProviderID) string {
	if err == nil {
		return "Invalid API key. Please try again."
	}

	errStr := err.Error()

	// Network errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout") {
		return "Connection failed. Check your internet and try again."
	}

	// Auth errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		switch provider {
		case llm.ProviderAnthropic:
			return "Invalid key. Verify at console.anthropic.com"
		case llm.ProviderOpenAI:
			return "Invalid key. Verify at platform.openai.com"
		case llm.ProviderGemini:
			return "Invalid key. Verify at aistudio.google.com"
		default:
			return "Authentication failed. Check your API key."
		}
	}

	// Rate limit
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate") {
		return "Rate limited. Wait a moment and try again."
	}

	// Truncate long errors
	if len(errStr) > 60 {
		return errStr[:57] + "..."
	}

	return errStr
}

func (m WizardModel) updateProviderSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.providerCursor > 0 {
			m.providerCursor--
		}
	case tea.KeyDown:
		if m.providerCursor < len(m.providerList)-1 {
			m.providerCursor++
		}
	case tea.KeyEnter:
		m.selectedProvider = m.providerList[m.providerCursor].id

		// Build auth methods for selected provider
		m.authMethods = nil
		m.authMethodCursor = 0
		for _, method := range auth.GetProviderAuthInfo(m.selectedProvider).Methods {
			m.authMethods = append(m.authMethods, authMethodItem{
				authType:    method.Type,
				label:       method.Label,
				description: method.Description,
			})
		}

		// If only one auth method (API key), skip selection
		if len(m.authMethods) == 1 {
			m.selectedAuth = m.authMethods[0].authType
			m.apiKeyInput.Focus()
			m.step = StepProviderKey
		} else {
			m.step = StepAuthMethod
		}
	}
	return m, nil
}

func (m WizardModel) updateAuthMethod(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.authMethodCursor > 0 {
			m.authMethodCursor--
		}
	case tea.KeyDown:
		if m.authMethodCursor < len(m.authMethods)-1 {
			m.authMethodCursor++
		}
	case tea.KeyEnter:
		m.selectedAuth = m.authMethods[m.authMethodCursor].authType
		if m.selectedAuth == "oauth" {
			m.oauthInProgress = true
			m.oauthError = ""
			m.step = StepOAuthWaiting
			return m, m.startOAuthFlow()
		}
		m.apiKeyInput.Focus()
		m.step = StepProviderKey
	}
	return m, nil
}

func (m WizardModel) updateProviderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.validatingKey {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		key := m.apiKeyInput.Value()
		if key == "" {
			m.keyError = "API key is required"
			return m, nil
		}
		m.validatingKey = true
		m.keyError = ""
		return m, m.validateKey()
	}
	return m, nil
}

func (m WizardModel) updateWalletChoice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.walletCursor > 0 {
			m.walletCursor--
		}
	case tea.KeyDown:
		if m.walletCursor < len(m.walletChoices)-1 {
			m.walletCursor++
		}
	case tea.KeyEnter:
		m.walletChoice = m.walletChoices[m.walletCursor]
		switch m.walletCursor {
		case 0: // Create new wallet
			m.passwordInput.Focus()
			m.step = StepWalletPassword
			m.passwordStep = 0
		case 1: // Import - coming soon
			m.passwordError = "Import wallet coming soon. Choose another option."
			return m, nil
		case 2: // Skip
			m.step = StepComplete
		}
	}
	return m, nil
}

func (m WizardModel) updateWalletPassword(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.passwordStep == 0 {
			if len(m.passwordInput.Value()) < 8 {
				m.passwordError = "Password must be at least 8 characters"
				return m, nil
			}
			m.passwordStep = 1
			m.passwordError = ""
			m.confirmInput.Focus()
		} else {
			if m.passwordInput.Value() != m.confirmInput.Value() {
				m.passwordError = "Passwords do not match. Try again."
				m.confirmInput.Reset()
				m.confirmInput.Focus()
				return m, nil
			}
			return m, m.createWallet()
		}
	}
	return m, nil
}

// View renders the wizard
func (m WizardModel) View() string {
	if m.quitting {
		if m.result != nil && m.result.Cancelled {
			return DimStyle.Render("\n  Setup cancelled.\n\n")
		}
		return ""
	}

	var b strings.Builder

	// Add progress bar for all steps except welcome and complete
	if m.step > StepWelcome && m.step < StepComplete {
		b.WriteString("\n")
		b.WriteString(m.renderProgress())
		b.WriteString("\n")
	}

	switch m.step {
	case StepWelcome:
		b.WriteString(m.viewWelcome())
	case StepProviderSelect:
		b.WriteString(m.viewProviderSelect())
	case StepAuthMethod:
		b.WriteString(m.viewAuthMethod())
	case StepProviderKey:
		b.WriteString(m.viewProviderKey())
	case StepOAuthWaiting:
		b.WriteString(m.viewOAuthWaiting())
	case StepWalletChoice:
		b.WriteString(m.viewWalletChoice())
	case StepWalletPassword:
		b.WriteString(m.viewWalletPassword())
	case StepComplete:
		b.WriteString(m.viewComplete())
	}

	return b.String()
}

func (m WizardModel) renderProgress() string {
	var currentStep int
	switch m.step {
	case StepProviderSelect, StepAuthMethod, StepProviderKey, StepOAuthWaiting:
		currentStep = 1
	case StepWalletChoice, StepWalletPassword, StepWalletConfirm:
		currentStep = 2
	case StepComplete:
		currentStep = 3
	}

	percent := float64(currentStep) / float64(totalSteps)
	bar := m.progress.ViewAs(percent)

	labels := "  Provider      Wallet       Ready"
	return fmt.Sprintf("  %s\n%s", bar, DimStyle.Render(labels))
}

func (m WizardModel) viewWelcome() string {
	var b strings.Builder
	b.WriteString("\n\n")

	// Check for detected env key
	if m.envKeyDetected {
		envVar := llm.EnvVarForProvider(m.envKeyProvider)
		providerName := string(m.envKeyProvider)
		for _, p := range m.providerList {
			if p.id == m.envKeyProvider {
				providerName = p.name
				break
			}
		}

		box := BoxStyle.Render(
			TitleStyle.Render("Welcome to clifi") + "\n" +
				SubtitleStyle.Render("Terminal-first crypto operator agent") + "\n\n" +
				SuccessStyle.Render(fmt.Sprintf("✓ Found %s in environment!", envVar)) + "\n" +
				fmt.Sprintf("  Using: %s", providerName),
		)
		b.WriteString(box)
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("  Press Enter to continue with detected key..."))
	} else {
		box := BoxStyle.Render(
			TitleStyle.Render("Welcome to clifi") + "\n" +
				SubtitleStyle.Render("Terminal-first crypto operator agent") + "\n\n" +
				"Let's get you set up in about 2 minutes.",
		)
		b.WriteString(box)
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("  Press Enter to continue..."))
	}

	return b.String()
}

func (m WizardModel) viewProviderSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(TitleStyle.Render("  Choose an LLM Provider"))
	b.WriteString("\n\n")
	b.WriteString(SubtitleStyle.Render("  clifi uses AI to understand your requests.\n"))
	b.WriteString("\n")

	for i, p := range m.providerList {
		cursor := "  "
		if i == m.providerCursor {
			cursor = CursorStyle.Render("> ")
		}

		name := p.name
		desc := DimStyle.Render(fmt.Sprintf(" - %s", p.description))

		if i == m.providerCursor {
			name = SelectedStyle.Render(name)
		} else {
			name = NormalStyle.Render(name)
		}

		suffix := ""
		if p.recommended {
			suffix = SuccessStyle.Render(" ★")
		}

		b.WriteString(fmt.Sprintf("%s%s%s%s\n", cursor, name, suffix, desc))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  ↑/↓ move • Enter select • Esc back"))
	return b.String()
}

func (m WizardModel) viewAuthMethod() string {
	var b strings.Builder
	b.WriteString("\n")

	providerName := string(m.selectedProvider)
	for _, p := range m.providerList {
		if p.id == m.selectedProvider {
			providerName = p.name
			break
		}
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("  How do you want to connect to %s?", providerName)))
	b.WriteString("\n\n")

	for i, method := range m.authMethods {
		cursor := "  "
		if i == m.authMethodCursor {
			cursor = CursorStyle.Render("> ")
		}

		label := method.label
		desc := DimStyle.Render(fmt.Sprintf(" - %s", method.description))

		if i == m.authMethodCursor {
			label = SelectedStyle.Render(label)
		} else {
			label = NormalStyle.Render(label)
		}

		b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, label, desc))
	}

	if m.oauthError != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", ErrorStyle.Render("✗ "+m.oauthError)))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  ↑/↓ move • Enter select • Esc back"))
	return b.String()
}

func (m WizardModel) viewOAuthWaiting() string {
	var b strings.Builder
	b.WriteString("\n")

	providerName := string(m.selectedProvider)
	for _, p := range m.providerList {
		if p.id == m.selectedProvider {
			providerName = p.name
			break
		}
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("  Connecting to %s", providerName)))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s Opening browser for authentication...\n\n", m.spinner.View()))
	b.WriteString(DimStyle.Render("  Complete the login in your browser.\n"))
	b.WriteString(DimStyle.Render("  Waiting for callback... (timeout: 5 minutes)\n"))

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  Esc to cancel"))
	return b.String()
}

func (m WizardModel) viewProviderKey() string {
	var b strings.Builder
	b.WriteString("\n")

	providerName := string(m.selectedProvider)
	for _, p := range m.providerList {
		if p.id == m.selectedProvider {
			providerName = p.name
			break
		}
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("  Enter %s API Key", providerName)))
	b.WriteString("\n\n")

	// Show where to get API key
	var apiUrl string
	switch m.selectedProvider {
	case llm.ProviderAnthropic:
		apiUrl = "console.anthropic.com"
	case llm.ProviderOpenAI:
		apiUrl = "platform.openai.com/api-keys"
	case llm.ProviderGemini:
		apiUrl = "aistudio.google.com/apikey"
	case llm.ProviderVenice:
		apiUrl = "venice.ai"
	case llm.ProviderCopilot:
		apiUrl = "Run: gh auth token"
	case llm.ProviderOpenRouter:
		apiUrl = "openrouter.ai/settings/keys"
	}
	b.WriteString(SubtitleStyle.Render(fmt.Sprintf("  Get your key at: %s\n\n", apiUrl)))

	// API key input using textinput
	b.WriteString("  ")
	b.WriteString(m.apiKeyInput.View())
	b.WriteString("\n")

	if m.validatingKey {
		b.WriteString(fmt.Sprintf("\n  %s Testing connection...\n", m.spinner.View()))
	} else if m.keyError != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", ErrorStyle.Render("✗ "+m.keyError)))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  Enter to validate • Esc back"))
	return b.String()
}

func (m WizardModel) viewWalletChoice() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(TitleStyle.Render("  Set Up Wallet"))
	b.WriteString(SubtitleStyle.Render(" (optional)"))
	b.WriteString("\n\n")

	// Explain what wallet does
	b.WriteString(DimStyle.Render("  A wallet lets you:\n"))
	b.WriteString(DimStyle.Render("  • Check balances across chains\n"))
	b.WriteString(DimStyle.Render("  • Send and receive crypto\n"))
	b.WriteString(DimStyle.Render("  • Interact with DeFi protocols\n"))
	b.WriteString("\n")

	for i, choice := range m.walletChoices {
		cursor := "  "
		if i == m.walletCursor {
			cursor = CursorStyle.Render("> ")
		}

		text := choice
		switch { //nolint:staticcheck // Can't use tagged switch with dynamic m.walletCursor
		case i == m.walletCursor:
			text = SelectedStyle.Render(text)
		case i == 1:
			// Import option is disabled
			text = DimStyle.Render(text)
		default:
			text = NormalStyle.Render(text)
		}

		b.WriteString(fmt.Sprintf("%s%s\n", cursor, text))
	}

	if m.passwordError != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", ErrorStyle.Render("✗ "+m.passwordError)))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  ↑/↓ move • Enter select • Esc back"))
	return b.String()
}

func (m WizardModel) viewWalletPassword() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(TitleStyle.Render("  Create Wallet Password"))
	b.WriteString("\n\n")

	b.WriteString(DimStyle.Render("  This encrypts your wallet on disk.\n"))
	b.WriteString(DimStyle.Render("  Requirements: 8+ characters\n\n"))

	if m.passwordStep == 0 {
		b.WriteString("  ")
		b.WriteString(m.passwordInput.View())
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("  Password: %s\n\n", SuccessStyle.Render("✓ set")))
		b.WriteString("  ")
		b.WriteString(m.confirmInput.View())
		b.WriteString("\n")
	}

	if m.passwordError != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", ErrorStyle.Render("✗ "+m.passwordError)))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  Enter to continue • Esc back"))
	return b.String()
}

func (m WizardModel) viewComplete() string {
	var b strings.Builder
	b.WriteString("\n\n")

	providerName := string(m.selectedProvider)
	for _, p := range m.providerList {
		if p.id == m.selectedProvider {
			providerName = p.name
			break
		}
	}

	walletInfo := DimStyle.Render("Not configured")
	if m.walletAddress != "" {
		short := m.walletAddress
		if len(short) > 10 {
			short = short[:6] + "..." + short[len(short)-4:]
		}
		walletInfo = short
	}

	content := fmt.Sprintf(
		"%s\n\n"+
			"Provider: %s\n"+
			"Wallet:   %s\n\n"+
			"%s\n"+
			"  %s\n"+
			"  %s\n"+
			"  %s",
		TitleStyle.Render("✨ You're all set!"),
		providerName,
		walletInfo,
		DimStyle.Render("Try these:"),
		"\"What's my portfolio?\"",
		"\"Show ETH balance on Base\"",
		"\"What chains are supported?\"",
	)

	b.WriteString(BoxStyle.Render(content))
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  Press Enter to start clifi..."))
	return b.String()
}

// RunWizard runs the setup wizard and returns the result
func RunWizard() (*SetupResult, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, err
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	m := NewWizard(dataDir)

	// Check if already fully configured
	if m.step == StepComplete && m.status.HasProvider {
		return &SetupResult{
			ProviderID:    m.selectedProvider,
			WalletCreated: m.status.HasWallet,
			WalletAddress: m.status.WalletAddress,
		}, nil
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(WizardModel).result
	return result, nil
}

// PrintEnvInstructions prints setup instructions for non-interactive environments
func PrintEnvInstructions() {
	fmt.Println("clifi requires an LLM provider to function.")
	fmt.Println("")
	fmt.Println("Set one of these environment variables:")
	fmt.Println("  ANTHROPIC_API_KEY=sk-ant-...")
	fmt.Println("  OPENAI_API_KEY=sk-...")
	fmt.Println("  GOOGLE_API_KEY=...")
	fmt.Println("  GITHUB_TOKEN=...")
	fmt.Println("  VENICE_API_KEY=...")
	fmt.Println("  OPENROUTER_API_KEY=...")
	fmt.Println("")
	fmt.Println("Or run clifi interactively to complete guided setup.")
}

// IsInteractive returns true if running in a terminal
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
