//go:build integration
// +build integration

package cli

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/agent"
	"github.com/yolodolo42/clifi/internal/llm"
)

// ScenarioStep represents a REPL action and expectation.
type ScenarioStep struct {
	Input       string   // user types (includes /commands)
	ExpectSubs  []string // substrings that must appear in output pane
	ExpectError bool     // whether the last line is an error-kind message
}

// Scenario describes a multi-step REPL interaction.
type Scenario struct {
	Name          string
	Provider      llm.ProviderID
	Model         string
	Steps         []ScenarioStep
	RequiresTools bool // if true, skip when model lacks tools
}

func TestScenario_OpenRouter_ToolUnsupportedIsGraceful(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}

	scenario := Scenario{
		Name:     "openrouter-claude-no-tools",
		Provider: llm.ProviderOpenRouter,
		Model:    "anthropic/claude-sonnet-4",
		Steps: []ScenarioStep{
			{
				Input:      "/status",
				ExpectSubs: []string{"Provider: openrouter", "Model: anthropic/claude-sonnet-4"},
			},
			{
				Input:       "list wallets", // will attempt a tool call and should be handled
				ExpectSubs:  []string{"Tools disabled for model", "openai/gpt-4o"},
				ExpectError: false,
			},
		},
		RequiresTools: false,
	}

	runScenario(t, scenario, key)
}

func TestScenario_OpenRouter_ToolHappyPath(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}

	scenario := Scenario{
		Name:     "openrouter-gpt4o-tools",
		Provider: llm.ProviderOpenRouter,
		Model:    "openai/gpt-4o",
		Steps: []ScenarioStep{
			{
				Input:      "/status",
				ExpectSubs: []string{"Provider: openrouter", "Model: openai/gpt-4o"},
			},
			{
				Input:       "list wallets",
				ExpectSubs:  []string{"wallet", "0x"},
				ExpectError: false,
			},
		},
		RequiresTools: true,
	}

	runScenario(t, scenario, key)
}

// runScenario spins up a REPL model with an Agent wired to OpenRouter.
func runScenario(t *testing.T, scenario Scenario, apiKey string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m := newModel()
	m.width = 100
	m.height = 40
	m.spinner = nil

	// Build auth manager and provider
	dataDir := getDataDir()
	authManager, err := getAuthManager()
	require.NoError(t, err)
	require.NoError(t, authManager.SetAPIKey(scenario.Provider, apiKey))
	require.NoError(t, authManager.SetDefaultProvider(scenario.Provider))

	a, err := agent.New(string(scenario.Provider))
	require.NoError(t, err)
	require.NoError(t, a.SetModel(scenario.Model))

	// If tools are required but model doesn't support tools, skip.
	supportsTools := modelSupportsTools(a, scenario.Model)
	if scenario.RequiresTools && !supportsTools {
		t.Skipf("model %s does not support tools", scenario.Model)
	}

	m.agent = a
	m.ready = true

	prog := bubbletea.NewProgram(m, bubbletea.WithContext(ctx))

	go func() {
		// drive steps
		for _, step := range scenario.Steps {
			prog.Send(bubbletea.KeyMsg{Type: bubbletea.KeyRunes, Runes: []rune(step.Input)})
			prog.Send(bubbletea.KeyMsg{Type: bubbletea.KeyEnter})
			time.Sleep(2 * time.Second)
		}
		// quit after steps
		prog.Send(bubbletea.KeyMsg{Type: bubbletea.KeyCtrlC})
	}()

	finalModel, err := prog.Run()
	require.NoError(t, err)

	mm := finalModel.(model)
	output := mm.viewport.View()

	for _, step := range scenario.Steps {
		for _, sub := range step.ExpectSubs {
			require.Contains(t, output, sub, "missing expected text for step %q", step.Input)
		}
		if step.ExpectError {
			require.True(t, strings.Contains(output, "Error") || strings.Contains(output, "error"), "expected error for step %q", step.Input)
		}
	}
}

func modelSupportsTools(a *agent.Agent, modelID string) bool {
	for _, m := range a.ListModels() {
		if m.ID == modelID {
			return m.SupportsTools
		}
	}
	return false
}
