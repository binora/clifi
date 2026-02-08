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
	"github.com/yolodolo42/clifi/internal/ui"
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
	StepComplete
)

const totalSteps = 3 // Provider, Wallet, Complete

var providerAPIKeyURLs = map[llm.ProviderID]string{
	llm.ProviderAnthropic:  "console.anthropic.com",
	llm.ProviderOpenAI:     "platform.openai.com/api-keys",
	llm.ProviderGemini:     "aistudio.google.com/apikey",
	llm.ProviderVenice:     "venice.ai",
	llm.ProviderCopilot:    "Run: gh auth token",
	llm.ProviderOpenRouter: "openrouter.ai/settings/keys",
}

var providerIDs = []llm.ProviderID{
	llm.ProviderAnthropic,
	llm.ProviderOpenAI,
	llm.ProviderGemini,
	llm.ProviderCopilot,
	llm.ProviderVenice,
	llm.ProviderOpenRouter,
}

var providerNames = map[llm.ProviderID]string{
	llm.ProviderAnthropic:  "Anthropic (Claude)",
	llm.ProviderOpenAI:     "OpenAI (GPT-4)",
	llm.ProviderGemini:     "Google (Gemini)",
	llm.ProviderCopilot:    "GitHub Copilot",
	llm.ProviderVenice:     "Venice AI",
	llm.ProviderOpenRouter: "OpenRouter",
}

var providerSelectorBaseItems = []ui.SelectorItem{
	{ID: string(llm.ProviderAnthropic), Label: providerNames[llm.ProviderAnthropic], Description: "recommended - Best reasoning & tool use"},
	{ID: string(llm.ProviderOpenAI), Label: providerNames[llm.ProviderOpenAI], Description: "Fast responses, widely used"},
	{ID: string(llm.ProviderGemini), Label: providerNames[llm.ProviderGemini], Description: "1M token context window"},
	{ID: string(llm.ProviderCopilot), Label: providerNames[llm.ProviderCopilot], Description: "Free with Copilot subscription"},
	{ID: string(llm.ProviderVenice), Label: providerNames[llm.ProviderVenice], Description: "Privacy-focused, uncensored"},
	{ID: string(llm.ProviderOpenRouter), Label: providerNames[llm.ProviderOpenRouter], Description: "Access 100+ models with one key"},
}

var walletSelectorItems = []ui.SelectorItem{
	{ID: "0", Label: "Create a new wallet"},
	{ID: "1", Label: "Import existing wallet (coming soon)", Description: "disabled"},
	{ID: "2", Label: "Continue without wallet"},
}

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
	quitting bool

	// Provider step
	providerList     []llm.ProviderID // kept for tests + env key scan ordering
	providerSelector ui.Selector
	selectedProvider llm.ProviderID
	apiKeyInput      textinput.Model
	validatingKey    bool
	keyError         string
	envKeyDetected   bool
	envKeyProvider   llm.ProviderID

	// Auth method step
	authSelector ui.Selector
	selectedAuth string // "api" or "oauth"
	oauthError   string

	// Wallet step
	walletChoices  []string // kept for tests
	walletSelector ui.Selector
	passwordInput  textinput.Model
	confirmInput   textinput.Model
	passwordStep   int // 0=enter, 1=confirm
	passwordError  string
	walletCreated  bool
	walletAddress  string

	// UI
	spinner  spinner.Model
	progress progress.Model

	// Result
	result *SetupResult
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

func withCurrent(items []ui.SelectorItem, id string) []ui.SelectorItem {
	out := make([]ui.SelectorItem, len(items))
	copy(out, items)
	for i := range out {
		out[i].Current = out[i].ID == id
	}
	return out
}

func secretInput(placeholder string, width, limit int) textinput.Model {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = placeholder
	in.EchoMode = textinput.EchoPassword
	in.EchoCharacter = '•'
	in.CharLimit = limit
	in.Width = width
	return in
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

	apiInput := secretInput("Paste your API key here...", 50, 200)
	passInput := secretInput("Enter password (8+ chars)", 40, 100)
	confirmInput := secretInput("Confirm password", 40, 100)

	currentProvider := ""
	if status.HasProvider {
		currentProvider = string(status.ProviderID)
	}
	providerSelector := ui.NewSelector("Choose an LLM provider", withCurrent(providerSelectorBaseItems, currentProvider))
	walletSelector := ui.NewSelector("Set up wallet (optional)", walletSelectorItems)

	m := &WizardModel{
		step:             StepWelcome,
		status:           status,
		dataDir:          dataDir,
		providerList:     providerIDs,
		providerSelector: providerSelector,
		walletChoices:    []string{walletSelectorItems[0].Label, walletSelectorItems[1].Label, walletSelectorItems[2].Label},
		walletSelector:   walletSelector,
		spinner:          sp,
		progress:         prog,
		apiKeyInput:      apiInput,
		passwordInput:    passInput,
		confirmInput:     confirmInput,
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
	for _, p := range providerIDs {
		envVar := llm.EnvVarForProvider(p)
		if envVar != "" && os.Getenv(envVar) != "" {
			m.envKeyDetected = true
			m.envKeyProvider = p
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
		// Global keys (don't swallow Esc; selectors use it).
		switch msg.Type {
		case tea.KeyCtrlC:
			m.result = &SetupResult{Cancelled: true}
			m.quitting = true
			return m, tea.Quit
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
			if msg.Type == tea.KeyEsc {
				m.apiKeyInput.Blur()
				m.apiKeyInput.Reset()
				m.keyError = ""
				// If we skipped auth selection (single method), go back to provider select.
				if len(auth.GetProviderAuthInfo(m.selectedProvider).Methods) <= 1 {
					m.step = StepProviderSelect
				} else {
					m.step = StepAuthMethod
				}
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				return m.updateProviderKey(msg)
			}
			// Fall through to let input update happen

		case StepOAuthWaiting:
			if msg.Type == tea.KeyEsc {
				m.oauthError = ""
				if len(auth.GetProviderAuthInfo(m.selectedProvider).Methods) <= 1 {
					m.step = StepProviderSelect
				} else {
					m.step = StepAuthMethod
				}
				return m, nil
			}
			// OAuth is in progress, just wait
			return m, nil

		case StepWalletChoice:
			return m.updateWalletChoice(msg)

		case StepWalletPassword:
			if msg.Type == tea.KeyEsc {
				m.passwordStep = 0
				m.passwordError = ""
				m.passwordInput.Reset()
				m.confirmInput.Reset()
				m.step = StepWalletChoice
				return m, nil
			}
			if msg.Type == tea.KeyEnter {
				return m.updateWalletPassword(msg)
			}
			// Fall through to let input update happen

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
		m.progress.Width = min(40, msg.Width-20)
		m.providerSelector.SetWidth(msg.Width)
		m.authSelector.SetWidth(msg.Width)
		m.walletSelector.SetWidth(msg.Width)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case keyValidatedMsg:
		m.validatingKey = false
		if msg.success {
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
		if apiURL := providerAPIKeyURLs[provider]; apiURL != "" {
			return "Invalid key. Verify at " + apiURL
		}
		return "Authentication failed. Check your API key."
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
	_, cmd := m.providerSelector.Update(msg)
	if cmd != nil {
		return m, cmd
	}

	if m.providerSelector.Active() {
		return m, nil
	}

	if m.providerSelector.Cancelled() {
		m.step = StepWelcome
		m.providerSelector = ui.NewSelector("Choose an LLM provider", withCurrent(providerSelectorBaseItems, ""))
		return m, nil
	}

	m.selectedProvider = llm.ProviderID(m.providerSelector.Selected())

	methods := auth.GetProviderAuthInfo(m.selectedProvider).Methods

	// If only one auth method (API key), skip selection
	if len(methods) == 1 {
		m.selectedAuth = methods[0].Type
		if m.selectedAuth == "oauth" {
			m.oauthError = ""
			m.step = StepOAuthWaiting
			return m, m.startOAuthFlow()
		}
		m.apiKeyInput.Focus()
		m.step = StepProviderKey
		return m, nil
	}

	items := make([]ui.SelectorItem, 0, len(methods))
	for _, method := range methods {
		items = append(items, ui.SelectorItem{ID: method.Type, Label: method.Label, Description: method.Description})
	}
	m.authSelector = ui.NewSelector("Choose authentication method", items)
	m.step = StepAuthMethod
	return m, nil
}

func (m WizardModel) updateAuthMethod(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	_, cmd := m.authSelector.Update(msg)
	if cmd != nil {
		return m, cmd
	}

	if m.authSelector.Active() {
		return m, nil
	}

	if m.authSelector.Cancelled() {
		m.step = StepProviderSelect
		m.providerSelector = ui.NewSelector("Choose an LLM provider", withCurrent(providerSelectorBaseItems, string(m.selectedProvider)))
		return m, nil
	}

	m.selectedAuth = m.authSelector.Selected()
	if m.selectedAuth == "oauth" {
		m.oauthError = ""
		m.step = StepOAuthWaiting
		return m, m.startOAuthFlow()
	}

	m.apiKeyInput.Focus()
	m.step = StepProviderKey
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
	_, cmd := m.walletSelector.Update(msg)
	if cmd != nil {
		return m, cmd
	}

	if m.walletSelector.Active() {
		return m, nil
	}

	if m.walletSelector.Cancelled() {
		m.step = StepProviderSelect
		m.walletSelector = ui.NewSelector("Set up wallet (optional)", walletSelectorItems)
		return m, nil
	}

	choice := m.walletSelector.Selected()
	switch choice {
	case "0": // create
		m.passwordInput.Focus()
		m.step = StepWalletPassword
		m.passwordStep = 0
		return m, nil
	case "1": // import disabled
		m.passwordError = "Import wallet coming soon. Choose another option."
		m.walletSelector = ui.NewSelector("Set up wallet (optional)", walletSelectorItems)
		return m, nil
	default: // skip
		m.step = StepComplete
		return m, nil
	}
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
	case StepWalletChoice, StepWalletPassword:
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
	if m.envKeyDetected {
		envVar := llm.EnvVarForProvider(m.envKeyProvider)
		providerName := m.providerName(m.envKeyProvider)

		box := BoxStyle.Render(TitleStyle.Render("Welcome to clifi") + "\n" + SubtitleStyle.Render("Terminal-first crypto operator agent") + "\n\n" +
			SuccessStyle.Render(fmt.Sprintf("✓ Found %s in environment!", envVar)) + "\n" + fmt.Sprintf("  Using: %s", providerName))
		return "\n\n" + box + "\n\n" + HelpStyle.Render("  Press Enter to continue with detected key...")
	}

	box := BoxStyle.Render(TitleStyle.Render("Welcome to clifi") + "\n" + SubtitleStyle.Render("Terminal-first crypto operator agent") + "\n\n" +
		"Let's get you set up in about 2 minutes.")
	return "\n\n" + box + "\n\n" + HelpStyle.Render("  Press Enter to continue...")
}

func (m WizardModel) viewProviderSelect() string {
	return "\n" + m.providerSelector.View()
}

func (m WizardModel) viewAuthMethod() string {
	errLine := ""
	if m.oauthError != "" {
		errLine = "\n" + ErrorStyle.Render("✗ "+m.oauthError) + "\n"
	}
	return "\n" + m.authSelector.View() + errLine
}

func (m WizardModel) viewOAuthWaiting() string {
	providerName := m.providerName(m.selectedProvider)
	return "\n" +
		TitleStyle.Render(fmt.Sprintf("  Connecting to %s", providerName)) + "\n\n" +
		fmt.Sprintf("  %s Opening browser for authentication...\n\n", m.spinner.View()) +
		DimStyle.Render("  Complete the login in your browser.\n  Waiting for callback... (timeout: 5 minutes)\n") + "\n" +
		HelpStyle.Render("  Esc to cancel")
}

func (m WizardModel) viewProviderKey() string {
	providerName := m.providerName(m.selectedProvider)

	header := "\n" + TitleStyle.Render(fmt.Sprintf("  Enter %s API Key", providerName)) + "\n\n"
	urlLine := ""
	if apiURL := providerAPIKeyURLs[m.selectedProvider]; apiURL != "" {
		urlLine = SubtitleStyle.Render(fmt.Sprintf("  Get your key at: %s\n\n", apiURL))
	}

	statusLine := ""
	if m.validatingKey {
		statusLine = fmt.Sprintf("\n  %s Testing connection...\n", m.spinner.View())
	} else if m.keyError != "" {
		statusLine = "\n  " + ErrorStyle.Render("✗ "+m.keyError) + "\n"
	}
	return header + urlLine + "  " + m.apiKeyInput.View() + "\n" + statusLine + "\n" + HelpStyle.Render("  Enter to validate • Esc back")
}

func (m WizardModel) viewWalletChoice() string {
	errLine := ""
	if m.passwordError != "" {
		errLine = "\n" + ErrorStyle.Render("✗ "+m.passwordError) + "\n"
	}
	return "\n" +
		DimStyle.Render("  A wallet lets you:\n  • Check balances across chains\n  • Send and receive crypto\n  • Interact with DeFi protocols\n\n") +
		m.walletSelector.View() + errLine
}

func (m WizardModel) viewWalletPassword() string {
	base := "\n" + TitleStyle.Render("  Create Wallet Password") + "\n\n" +
		DimStyle.Render("  This encrypts your wallet on disk.\n  Requirements: 8+ characters\n\n")
	if m.passwordStep == 0 {
		base += "  " + m.passwordInput.View() + "\n"
	} else {
		base += fmt.Sprintf("  Password: %s\n\n  %s\n", SuccessStyle.Render("✓ set"), m.confirmInput.View())
	}

	if m.passwordError != "" {
		base += "\n  " + ErrorStyle.Render("✗ "+m.passwordError) + "\n"
	}

	return base + "\n" + HelpStyle.Render("  Enter to continue • Esc back")
}

func (m WizardModel) viewComplete() string {
	var b strings.Builder
	b.WriteString("\n\n")

	providerName := m.providerName(m.selectedProvider)

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

func (m WizardModel) providerName(id llm.ProviderID) string {
	if name := providerNames[id]; name != "" {
		return name
	}
	return string(id)
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
