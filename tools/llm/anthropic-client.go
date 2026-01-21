package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

// AnthropicClient implements the Client interface for Claude
type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAnthropicClient creates a new Anthropic API client
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-sonnet-4-5"
	}
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// anthropicRequest is the API request structure
type anthropicRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system,omitempty"`
	Messages  []anthropicMsg `json:"messages"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the API response structure
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends a prompt to Claude and returns the response
func (c *AnthropicClient) Complete(ctx context.Context, systemPrompt string, messages []Message, opts *RequestOptions) (*Response, error) {
	start := time.Now()

	// Default to 16K tokens for detailed sketch generation
	maxTokens := 16384
	if opts != nil && opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}

	// Convert messages to Anthropic format
	anthropicMsgs := make([]anthropicMsg, len(messages))
	for i, m := range messages {
		anthropicMsgs[i] = anthropicMsg{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  anthropicMsgs,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			return nil, fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text content
	var content string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &Response{
		Content:      content,
		InputTokens:  apiResp.Usage.InputTokens,
		OutputTokens: apiResp.Usage.OutputTokens,
		Duration:     time.Since(start),
		Model:        apiResp.Model,
		StopReason:   apiResp.StopReason,
	}, nil
}

// CompleteWithRetry attempts completion with retries on failure
func (c *AnthropicClient) CompleteWithRetry(ctx context.Context, systemPrompt string, messages []Message, maxRetries int, opts *RequestOptions) (*Response, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := c.Complete(ctx, systemPrompt, messages, opts)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Exponential backoff
		backoff := time.Duration(1<<uint(i)) * time.Second
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}