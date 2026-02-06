package agent

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yolodolo42/clifi/internal/llm"
)

// Canonical live scenarios. Skips when env keys are missing.
func TestCanonical_OpenRouter_Claude_ListWallets(t *testing.T) {
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	ag, err := New("")
	require.NoError(t, err)
	defer ag.Close()

	require.NoError(t, ag.SetProvider(llm.ProviderOpenRouter))
	require.NoError(t, ag.SetModel("anthropic/claude-3.5-sonnet"))

	events, err := ag.ChatWithEvents(ctx, "list wallets")
	require.NoError(t, err)
	require.NotEmpty(t, events)
}
