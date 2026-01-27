package agent

import (
	"encoding/json"
	"time"

	"github.com/yolodolo42/clifi/internal/llm"
)

// ConversationTurn represents a single turn in a conversation
type ConversationTurn struct {
	Timestamp  time.Time       `json:"timestamp"`
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []llm.ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *llm.ToolResult `json:"tool_result,omitempty"`
}

// Conversation holds the full conversation state
type Conversation struct {
	ID        string             `json:"id"`
	StartedAt time.Time          `json:"started_at"`
	Turns     []ConversationTurn `json:"turns"`
}

// NewConversation creates a new conversation
func NewConversation() *Conversation {
	return &Conversation{
		ID:        generateID(),
		StartedAt: time.Now(),
		Turns:     make([]ConversationTurn, 0),
	}
}

// AddUserMessage adds a user message to the conversation
func (c *Conversation) AddUserMessage(content string) {
	c.Turns = append(c.Turns, ConversationTurn{
		Timestamp: time.Now(),
		Role:      "user",
		Content:   content,
	})
}

// AddAssistantMessage adds an assistant message to the conversation
func (c *Conversation) AddAssistantMessage(content string, toolCalls []llm.ToolCall) {
	c.Turns = append(c.Turns, ConversationTurn{
		Timestamp: time.Now(),
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
}

// AddToolResult adds a tool result to the conversation
func (c *Conversation) AddToolResult(result llm.ToolResult) {
	c.Turns = append(c.Turns, ConversationTurn{
		Timestamp:  time.Now(),
		Role:       "tool",
		ToolResult: &result,
	})
}

// ToMessages converts the conversation to LLM messages format
func (c *Conversation) ToMessages() []llm.Message {
	messages := make([]llm.Message, 0)
	for _, turn := range c.Turns {
		if turn.Role == "user" || turn.Role == "assistant" {
			messages = append(messages, llm.Message{
				Role:    turn.Role,
				Content: turn.Content,
			})
		}
	}
	return messages
}

// ToJSON serializes the conversation to JSON
func (c *Conversation) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// generateID creates a simple unique ID for the conversation
func generateID() string {
	return time.Now().Format("20060102-150405")
}
