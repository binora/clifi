package llm

import "encoding/json"

func NewTool(name, description string, schema any) Tool {
	schemaBytes, _ := json.Marshal(schema)
	return Tool{
		Name:        name,
		Description: description,
		InputSchema: schemaBytes,
	}
}
