package llm

import (
	"context"
	"time"
)

// Message represents a chat message
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// Response represents an LLM response
type Response struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	Model        string
}

// Client defines the interface for LLM interactions
type Client interface {
	// Complete sends a prompt and returns a response
	Complete(ctx context.Context, systemPrompt string, messages []Message) (*Response, error)

	// CompleteWithRetry attempts completion with retries on failure
	CompleteWithRetry(ctx context.Context, systemPrompt string, messages []Message, maxRetries int) (*Response, error)
}
