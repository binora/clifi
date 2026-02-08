package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactJSONArgs(t *testing.T) {
	got := RedactJSONArgs(`{"password":"pw","nested":{"access_token":"tok","keep":1},"arr":[{"secret":"s"}]}`)
	require.Contains(t, got, `"password":"***REDACTED***"`)
	require.Contains(t, got, `"access_token":"***REDACTED***"`)
	require.Contains(t, got, `"secret":"***REDACTED***"`)
	require.Contains(t, got, `"keep":1`)
}
