package testutil

import (
	"os"
	"testing"
)

// TempDir creates a temporary directory for testing and registers cleanup
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "clifi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir) // Best-effort cleanup
	})
	return dir
}

// SetEnv sets an environment variable and restores it after the test
func SetEnv(t *testing.T, key, value string) {
	t.Helper()
	old, hadOld := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env var %s: %v", key, err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

// UnsetEnv unsets an environment variable and restores it after the test
func UnsetEnv(t *testing.T, key string) {
	t.Helper()
	old, hadOld := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset env var %s: %v", key, err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
		}
	})
}
