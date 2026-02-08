package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type sessionLogger struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

func newSessionLogger(dataDir, sessionID string) (*sessionLogger, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data dir not configured")
	}
	dir := filepath.Join(dataDir, "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, sessionID+".jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	return &sessionLogger{path: path, f: f}, nil
}

func (l *sessionLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		_ = l.f.Close()
		l.f = nil
	}
}

func (l *sessionLogger) logRecord(v any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return
	}

	// One JSON object per line to keep it append-only and streamable.
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	b = append(b, '\n')
	_, _ = l.f.Write(b)
}

type sessionRecord struct {
	TS   string `json:"ts"`
	Type string `json:"type"`

	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`

	Content string `json:"content,omitempty"`

	ToolName string    `json:"tool_name,omitempty"`
	Args     string    `json:"args,omitempty"`
	Text     string    `json:"text,omitempty"`
	Blocks   []UIBlock `json:"blocks,omitempty"`
	IsError  bool      `json:"is_error,omitempty"`
}

func nowTS() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
