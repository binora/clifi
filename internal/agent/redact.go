package agent

import (
	"encoding/json"
	"strings"
)

var redactKeys = map[string]struct{}{
	"password":      {},
	"api_key":       {},
	"apikey":        {},
	"access_token":  {},
	"refresh_token": {},
	"private_key":   {},
	"secret":        {},
}

func RedactJSONArgs(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}

	redacted := redactValue(v)
	b, err := json.Marshal(redacted)
	if err != nil {
		return raw
	}
	return string(b)
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			if _, ok := redactKeys[strings.ToLower(k)]; ok {
				out[k] = "***REDACTED***"
				continue
			}
			out[k] = redactValue(vv)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = redactValue(t[i])
		}
		return out
	default:
		return v
	}
}
