package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LLMClient interface {
	Complete(system string, messages []Message) (string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic client
type AnthropicClient struct {
	key string
	log *Logger
}

func NewAnthropicClient(key string, log *Logger) *AnthropicClient {
	return &AnthropicClient{key: key, log: log}
}

func (c *AnthropicClient) Complete(system string, messages []Message) (string, error) {
	body := map[string]any{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 16384,
		"system":     system,
		"messages":   messages,
	}

	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.key)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	c.log.Debug("received %d chars", len(result.Content[0].Text))
	return result.Content[0].Text, nil
}

// Local LMStudio client (OpenAI-compatible)
type LocalClient struct {
	log *Logger
}

func NewLocalClient(log *Logger) *LocalClient {
	return &LocalClient{log: log}
}

func (c *LocalClient) Complete(system string, messages []Message) (string, error) {
	msgs := []Message{{Role: "system", Content: system}}
	msgs = append(msgs, messages...)

	body := map[string]any{
		"messages":   msgs,
		"max_tokens": 16384,
	}

	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:1234/v1/chat/completions", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LMStudio connection failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	c.log.Debug("received %d chars", len(result.Choices[0].Message.Content))
	return result.Choices[0].Message.Content, nil
}