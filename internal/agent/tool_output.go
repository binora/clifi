package agent

// ToolOutput is the dual-channel tool response:
// - Text: what we return to the LLM (and what users can copy/paste)
// - Blocks: structured UI payload for the REPL to render without parsing text
type ToolOutput struct {
	Text   string    `json:"text"`
	Blocks []UIBlock `json:"blocks,omitempty"`
}
