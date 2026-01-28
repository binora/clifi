package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWizard_InputPrompts(t *testing.T) {
	t.Run("all textinputs have empty prompt", func(t *testing.T) {
		m := NewWizard("")

		// Verify all textinputs have empty prompt (no "> " prefix)
		assert.Equal(t, "", m.apiKeyInput.Prompt, "apiKeyInput should have empty prompt")
		assert.Equal(t, "", m.passwordInput.Prompt, "passwordInput should have empty prompt")
		assert.Equal(t, "", m.confirmInput.Prompt, "confirmInput should have empty prompt")
	})
}

func TestNewWizard_Initialization(t *testing.T) {
	t.Run("initializes with StepWelcome", func(t *testing.T) {
		m := NewWizard("")
		assert.Equal(t, StepWelcome, m.step)
	})

	t.Run("has provider list", func(t *testing.T) {
		m := NewWizard("")
		assert.Greater(t, len(m.providerList), 0, "should have providers")
	})

	t.Run("has wallet choices", func(t *testing.T) {
		m := NewWizard("")
		assert.Greater(t, len(m.walletChoices), 0, "should have wallet choices")
	})
}
