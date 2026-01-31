package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ToolCapabilitiesCache caches per-provider model capability lookups.
type ToolCapabilitiesCache struct {
	mu      sync.Mutex
	entries map[ProviderID]capEntry
}

type capEntry struct {
	expiry  time.Time
	support map[string]bool // modelID -> supportsTools
}

var toolCapCache = &ToolCapabilitiesCache{
	entries: make(map[ProviderID]capEntry),
}

// SupportsToolsForModel returns (supports, known) for a provider/model.
// known==false means we could not determine and callers may choose to fallback.
func SupportsToolsForModel(ctx context.Context, provider Provider, modelID string, openRouterAPIKey string) (bool, bool) {
	// Prefer the provider's static model list.
	for _, m := range provider.Models() {
		if m.ID == modelID {
			return m.SupportsTools, true
		}
	}

	// Dynamic fetch for OpenRouter to avoid stale model lists.
	if provider.ID() == ProviderOpenRouter {
		if supports, known := toolCapCache.fetchOpenRouter(ctx, openRouterAPIKey, modelID); known {
			return supports, true
		}
	}

	return true, false // default optimistic
}

func (c *ToolCapabilitiesCache) fetchOpenRouter(ctx context.Context, apiKey, targetModel string) (bool, bool) {
	if apiKey == "" {
		return false, false
	}

	c.mu.Lock()
	entry, ok := c.entries[ProviderOpenRouter]
	now := time.Now()
	if ok && now.Before(entry.expiry) {
		if v, found := entry.support[targetModel]; found {
			c.mu.Unlock()
			return v, true
		}
	}
	c.mu.Unlock()

	// Refresh
	support, err := pullOpenRouterModels(ctx, apiKey)
	if err != nil {
		return false, false
	}

	c.mu.Lock()
	c.entries[ProviderOpenRouter] = capEntry{
		expiry:  time.Now().Add(6 * time.Hour),
		support: support,
	}
	v, found := support[targetModel]
	c.mu.Unlock()

	if found {
		return v, true
	}
	return false, false
}

func pullOpenRouterModels(ctx context.Context, apiKey string) (map[string]bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var body struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make(map[string]bool)
	for _, raw := range body.Data {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			continue
		}
		if supportsToolsInOpenRouter(m) {
			out[id] = true
		} else {
			out[id] = false
		}
	}
	return out, nil
}

func supportsToolsInOpenRouter(m map[string]any) bool {
	// supported_parameters array check
	if arr, ok := m["supported_parameters"]; ok {
		if hasToolish(arr) {
			return true
		}
	}
	// top_provider.supported_parameters
	if tp, ok := m["top_provider"].(map[string]any); ok {
		if arr, ok := tp["supported_parameters"]; ok && hasToolish(arr) {
			return true
		}
	}
	// capabilities.tools / function_calling
	if caps, ok := m["capabilities"].(map[string]any); ok {
		for _, key := range []string{"tools", "function_calling", "functions"} {
			if b, ok := caps[key].(bool); ok && b {
				return true
			}
		}
	}
	return false
}

func hasToolish(v any) bool {
	arr, ok := v.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.ToLower(s)
		if strings.Contains(s, "tool") || strings.Contains(s, "function") {
			return true
		}
	}
	return false
}
