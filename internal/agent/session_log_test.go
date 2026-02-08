package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionLogger_WritesJSONLAndPermissions(t *testing.T) {
	dir := t.TempDir()
	l, err := newSessionLogger(dir, "test-session")
	require.NoError(t, err)
	t.Cleanup(l.Close)

	l.logRecord(sessionRecord{TS: nowTS(), Type: "user", Content: "hi"})
	l.logRecord(sessionRecord{TS: nowTS(), Type: "tool_call", ToolName: "send_native", Args: RedactJSONArgs(`{"password":"pw"}`)})

	path := filepath.Join(dir, "sessions", "test-session.jsonl")
	st, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), st.Mode().Perm())

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(b), `"type":"user"`)
	require.Contains(t, string(b), `"type":"tool_call"`)
	require.Contains(t, string(b), `***REDACTED***`)
}
