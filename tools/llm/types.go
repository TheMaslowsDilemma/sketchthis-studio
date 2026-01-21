package llm

import ( 
    "context"
	"time"
)

// Client is the interface for LLM providers
type Client interface {
	Complete(ctx context.Context, systemPrompt string, messages []Message, opts *RequestOptions) (*Response, error)
	CompleteWithRetry(ctx context.Context, systemPrompt string, messages []Message, maxRetries int, opts *RequestOptions) (*Response, error)
}

// Message represents a conversation message
type Message struct {
	Role    string
	Content string
}

// RequestOptions configures an LLM request
type RequestOptions struct {
	MaxTokens int
}

// Response from an LLM completion
type Response struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	Model        string
	StopReason   string // "end_turn", "max_tokens", "stop_sequence"
}

// WasTruncated returns true if the response hit the token limit
func (r *Response) WasTruncated() bool {
	return r.StopReason == "max_tokens"
}